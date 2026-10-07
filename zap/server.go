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
	}
	h := &handlers{manager: s.Manager}
	s.http = h.httpServer()
	s.https = h.httpsServer()

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

func (h *handlers) httpServer() *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/", h.ensureApp(h.app))

	return &http.Server{
		Handler: mux,
	}
}

func (h *handlers) httpsServer() *http.Server {
	mux := http.NewServeMux()
	// TODO: don't handle these requests unless localhost request (eg. not via ngrok)
	// Maybe have a zapHandler that checks for localhost and then delegates requests
	mux.HandleFunc("/zap/api/apps", h.appsAPI)
	mux.HandleFunc("/zap/api/log", h.findApp(h.logAPI))
	mux.HandleFunc("/zap/api/state", h.findApp(h.stateAPI))
	mux.HandleFunc("/zap/ngrok/start", h.findApp(h.startNgrok))
	mux.HandleFunc("/zap/ngrok", h.findApp(h.ngrok))
	mux.HandleFunc("/zap/log", h.findApp(h.log))
	mux.HandleFunc("/zap/restart", h.findApp(h.restart))
	mux.HandleFunc("/zap", h.ensureApp(h.status))
	mux.HandleFunc("/", h.ensureApp(h.app))

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
