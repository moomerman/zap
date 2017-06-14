package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"

	"github.com/moomerman/phx-dev/devcert"
)

func main() {
	cache, err := devcert.NewCertCache()
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
