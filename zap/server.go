package zap

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"syscall"
	"time"

	"github.com/moomerman/zap/cert"
	"golang.org/x/net/http2"
)

// Server holds the state for the HTTP and HTTPS servers
type Server struct {
	HTTPAddr  string
	HTTPSAddr string
	// LogDir is where app logs are written, see Manager.LogDir
	LogDir string

	// Manager runs the apps. Serve creates one if it isn't set.
	Manager *Manager

	http  *http.Server
	https *http.Server
}

// Serve starts the HTTP servers and blocks until they have stopped and every
// app has been shut down
func (s *Server) Serve() {
	if s.Manager == nil {
		s.Manager = NewManager()
		s.Manager.LogDir = s.LogDir
	}
	h := &proxyHandler{manager: s.Manager}
	s.http = &http.Server{Handler: h}
	s.https = httpsServer(h)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		if err := s.serveHTTP(); err != http.ErrServerClosed {
			log.Println("[zap] http server stopped unexpectedly", err)
		}
	}()

	go func() {
		defer wg.Done()
		if err := s.serveHTTPS(); err != http.ErrServerClosed {
			log.Println("[zap] https server stopped unexpectedly", err)
		}
	}()

	wg.Wait()

	// stop the apps once no more requests can start them
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := s.Manager.Shutdown(ctx); err != nil {
		log.Println("[zap] apps did not all stop", err)
	}
}

// Stop gracefully stops the HTTP and HTTPS servers. Serve then stops the
// apps before returning.
func (s *Server) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	s.http.Shutdown(ctx)
	s.https.Shutdown(ctx)
}

func (s *Server) serveHTTP() error {
	listener, err := listen(s.HTTPAddr)
	if err != nil {
		log.Fatal("[zap] unable to create listener ", err)
	}

	log.Println("[zap] http listening at", listener.Addr())
	return s.http.Serve(listener)
}

func (s *Server) serveHTTPS() error {
	listener, err := listen(s.HTTPSAddr)
	if err != nil {
		log.Fatal("[zap] unable to create tls listener ", err)
	}

	log.Println("[zap] https listening at", listener.Addr())
	return s.https.Serve(tls.NewListener(listener, s.https.TLSConfig))
}

func listen(addr string) (net.Listener, error) {
	// launchd socket activation is gone, so an old launch agent passing
	// -http=Socket needs replacing
	if addr == "Socket" || addr == "SocketTLS" {
		return nil, fmt.Errorf("%q is from an old launch agent, run zapd -install again", addr)
	}

	listener, err := net.Listen("tcp", addr)
	if !errors.Is(err, syscall.EACCES) {
		return listener, err
	}

	// macOS lets a normal user bind ports below 1024 on all interfaces but
	// not on 127.0.0.1, so listen everywhere and only serve this machine
	host, port, splitErr := net.SplitHostPort(addr)
	if splitErr != nil || !isLoopback(host) {
		return nil, fmt.Errorf("%w (this system needs root to listen on ports below 1024, use higher ports with -http and -https)", err)
	}
	listener, err = net.Listen("tcp", net.JoinHostPort("", port))
	if err != nil {
		return nil, err
	}
	log.Printf("[zap] %s needs root, listening on all interfaces at %s and refusing other machines\n", addr, listener.Addr())
	return &loopbackListener{listener}, nil
}

func isLoopback(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// loopbackListener closes connections that don't come from this machine
type loopbackListener struct {
	net.Listener
}

func (l *loopbackListener) Accept() (net.Conn, error) {
	for {
		conn, err := l.Listener.Accept()
		if err != nil {
			return nil, err
		}
		if addr, ok := conn.RemoteAddr().(*net.TCPAddr); ok && addr.IP.IsLoopback() {
			return conn, nil
		}
		conn.Close()
	}
}

func httpsServer(h http.Handler) *http.Server {
	cache, err := cert.NewCache()
	if err != nil {
		log.Fatal("[zap] unable to create new cert cache", err)
	}

	tlsConfig := &tls.Config{
		GetCertificate: cache.GetCertificate,
	}

	server := &http.Server{
		Handler:   h,
		TLSConfig: tlsConfig,
	}
	http2.ConfigureServer(server, nil)

	return server
}
