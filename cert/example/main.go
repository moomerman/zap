package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
)

func main() {
	cache, err := cert.NewCertCache()
	if err != nil {
		log.Fatal("Unable to create new cert cache", err)
	}

	tlsConfig := &tls.Config{
		GetCertificate: cache.GetCertificate,
	}

	server := &http.Server{
		TLSConfig: tlsConfig,
	}

	listener, err := tls.Listen("tcp", ":4443", tlsConfig)
	if err != nil {
		log.Fatal("Unable to create listener", err)
	}

	fmt.Println(server.Serve(listener))
}
