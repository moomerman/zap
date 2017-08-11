package dev

import (
	"crypto/tls"
	"encoding/json"
	"log"
	"net/http"

	"github.com/moomerman/zap/cert"
	"github.com/puma/puma-dev/dev/launch"
	"golang.org/x/net/http2"
)

// Server holds the state for the HTTP and HTTPS servers
type Server struct {
	http  *http.Server
	https *http.Server
}

// NewServer starts the HTTP and HTTPS proxy servers
func NewServer() *Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/zap/log", logHandler())
	mux.HandleFunc("/zap/status", statusHandler())
	mux.HandleFunc("/", appHandler())

	http := startHTTP(mux)
	https := startHTTPS(mux)

	return &Server{
		http:  http,
		https: https,
	}
}

// ServeTLS starts the HTTPS server
func (s *Server) ServeTLS(bind string) {
	if bind == "SocketTLS" {
		listeners, err := launch.SocketListeners(bind)
		if err != nil {
			log.Fatal("unable to get launchd socket listener", err)
		}

		s.https.Serve(tls.NewListener(listeners[0], s.https.TLSConfig))
	} else {
		listener, err := tls.Listen("tcp", bind, s.https.TLSConfig)
		if err != nil {
			log.Fatal("unable to create tls listener", err)
		}
		s.https.Serve(listener)
	}
}

// Serve starts the HTTP server
func (s *Server) Serve(bind string) {
}

func startHTTPS(handler http.Handler) *http.Server {
	cache, err := cert.NewCache()
	if err != nil {
		log.Fatal("unable to create new cert cache", err)
	}

	tlsConfig := &tls.Config{
		GetCertificate: cache.GetCertificate,
	}

	server := &http.Server{
		Handler:   handler,
		TLSConfig: tlsConfig,
	}
	http2.ConfigureServer(server, nil)

	return server
}

func startHTTP(handler http.Handler) *http.Server {
	return nil
}

func appHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		app, err := findAppForHost(r.Host)
		if err != nil {
			http.Error(w, "502 App Not Found", http.StatusBadGateway)
			return
		}

		app.ServeHTTP(w, r)
	}
}

func logHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		app, err := findAppForHost(r.Host)
		if err != nil {
			http.Error(w, "502 App Not Found", http.StatusBadGateway)
			return
		}

		app.WriteLog(w)
	}
}

func statusHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		app, err := findAppForHost(r.Host)
		if err != nil {
			http.Error(w, "502 App Not Found", http.StatusBadGateway)
			return
		}

		content, _ := json.MarshalIndent(app, "", "  ")

		w.Header().Set("Content-Type", "application/json")
		w.Write(content)
	}
}
