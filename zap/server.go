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
