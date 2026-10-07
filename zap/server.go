package zap

import (
	"context"
	"crypto/tls"
	"log"
	"net/http"
	"sync"
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
