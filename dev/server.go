package dev

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"

	"github.com/moomerman/phx-dev/devcert"
	"golang.org/x/net/http2"
)

// Start starts the HTTP and HTTPS proxy servers
func Start() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", proxyHandler())

	startHTTP(mux)
	startHTTPS(mux)
}

func startHTTPS(handler http.Handler) {
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

	listener, err := tls.Listen("tcp", ":443", tlsConfig)
	if err != nil {
		log.Fatal("unable to create listener", err)
	}

	fmt.Println(server.Serve(listener))
}

func startHTTP(handler http.Handler) {

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
