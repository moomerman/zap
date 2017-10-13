package zap

import (
	"context"
	"crypto/tls"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/moomerman/zap/cert"
	"github.com/puma/puma-dev/dev/launch"
	"golang.org/x/net/http2"
)

// Server holds the state for the HTTP and HTTPS servers
type Server struct {
	HTTPAddr  string
	HTTPSAddr string

	http  *http.Server
	https *http.Server
}

// Serve starts the HTTP servers
func (s *Server) Serve() {
	s.http = createHTTPServer()
	s.https = createHTTPSServer()

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
}

// Stop gracefully stops the HTTP and HTTPS servers
func (s *Server) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	s.http.Shutdown(ctx)
	s.https.Shutdown(ctx)
}

func (s *Server) serveHTTP() error {
	var listener net.Listener
	var err error

	if s.HTTPAddr == "Socket" {
		listener = getSocketListener(s.HTTPAddr)
	} else {
		listener, err = net.Listen("tcp", s.HTTPAddr)
		if err != nil {
			log.Fatal("[zap] unable to create listener", err)
		}
	}

	log.Println("[zap] http listening at", listener.Addr())
	return s.http.Serve(listener)
}

func (s *Server) serveHTTPS() error {
	var listener net.Listener
	var err error

	if s.HTTPSAddr == "SocketTLS" {
		listener = tls.NewListener(getSocketListener(s.HTTPSAddr), s.https.TLSConfig)
	} else {
		listener, err = tls.Listen("tcp", s.HTTPSAddr, s.https.TLSConfig)
		if err != nil {
			log.Fatal("[zap] unable to create tls listener", err)
		}
	}

	log.Println("[zap] https listening at", listener.Addr())
	return s.https.Serve(listener)
}

func createHTTPServer() *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/", findAppHandler(appHandler))

	return &http.Server{
		Handler: mux,
	}
}

func createHTTPSServer() *http.Server {
	mux := http.NewServeMux()
	// TODO: don't handle these requests unless localhost request (eg. not via ngrok)
	// Maybe have a zapHandler that checks for localhost and then delegates requests
	mux.HandleFunc("/zap/api/apps", appsAPIHandler)
	mux.HandleFunc("/zap/api/log", findAppHandler(logAPIHandler))
	mux.HandleFunc("/zap/api/state", findAppHandler(stateAPIHandler))
	mux.HandleFunc("/zap/ngrok/start", findAppHandler(startNgrokHandler))
	mux.HandleFunc("/zap/ngrok", findAppHandler(ngrokHandler))
	mux.HandleFunc("/zap/log", findAppHandler(logHandler))
	mux.HandleFunc("/zap/restart", findAppHandler(restartHandler))
	mux.HandleFunc("/zap", findAppHandler(statusHandler))
	mux.HandleFunc("/", findAppHandler(appHandler))

	cache, err := cert.NewCache()
	if err != nil {
		log.Fatal("[zap] unable to create new cert cache", err)
	}

	tlsConfig := &tls.Config{
		GetCertificate: cache.GetCertificate,
	}

	server := &http.Server{
		Handler:   mux,
		TLSConfig: tlsConfig,
	}
	http2.ConfigureServer(server, nil)

	return server
}

func getSocketListener(socket string) net.Listener {
	listeners, err := launch.SocketListeners(socket)
	if err != nil {
		log.Fatal("[zap] unable to get launchd socket listener", err)
	}
	return listeners[0]
}
