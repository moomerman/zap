package dev

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"

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
	certCache := NewCertCache()

	tlsConfig := &tls.Config{
		GetCertificate: certCache.GetCertificate,
	}

	server := &http.Server{
		Handler:   handler,
		TLSConfig: tlsConfig,
	}
	http2.ConfigureServer(server, nil)

	fmt.Println(" [StartSSL] starting")

	listener, err := tls.Listen("tcp", ":4443", tlsConfig)
	if err != nil {
		log.Fatal(" [StartSSL] exited")
	}

	fmt.Println(server.Serve(listener))
}

func startHTTP(handler http.Handler) {

}
