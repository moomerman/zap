package dev

import (
	"crypto/tls"
	"fmt"
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
	mux.HandleFunc("/", proxyHandler())

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

func proxyHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}

		app, err := findAppForHost(r.Host)
		if err != nil {
			// render an error
			return
		}

		source := fmt.Sprint(r.Method, " ", r.Proto, " ", scheme+"://", r.Host, r.URL)
		fmt.Println("[proxy]", source, "->", app.proxy.URL)
		app.proxy.Proxy(w, r)
	}
}
