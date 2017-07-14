package dev

import (
	"crypto/tls"
	"encoding/json"
	"log"
	"net/http"

	"github.com/moomerman/phx-dev/devcert"
	"github.com/puma/puma-dev/dev/launch"
	"golang.org/x/net/http2"
)

type Server struct {
	http  *http.Server
	https *http.Server
}

// NewServer starts the HTTP and HTTPS proxy servers
func NewServer() *Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/phx/log", logHandler())
	mux.HandleFunc("/phx/status", statusHandler())
	mux.HandleFunc("/", appHandler())

	http := startHTTP(mux)
	https := startHTTPS(mux)

	return &Server{
		http:  http,
		https: https,
	}
}

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

func (s *Server) Serve(bind string) {
}

func startHTTPS(handler http.Handler) *http.Server {
	cache, err := devcert.NewCertCache()
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

		app.Serve(w, r)
	}
}

func logHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		app, err := findAppForHost(r.Host)
		if err != nil {
			http.Error(w, "502 App Not Found", http.StatusBadGateway)
			return
		}

		app.log.WriteTo(w)
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
