package zap

import (
	"crypto/tls"
	"log"
	"net"
)

func (s *Server) serveHTTP() error {
	var listener net.Listener
	var err error

	listener, err = net.Listen("tcp", s.HTTPAddr)
	if err != nil {
		log.Fatal("[zap] unable to create listener", err)
	}

	log.Println("[zap] http listening at", listener.Addr())
	return s.http.Serve(listener)
}

func (s *Server) serveHTTPS() error {
	var listener net.Listener
	var err error

	listener, err = tls.Listen("tcp", s.HTTPSAddr, s.https.TLSConfig)
	if err != nil {
		log.Fatal("[zap] unable to create tls listener", err)
	}

	log.Println("[zap] https listening at", listener.Addr())
	return s.https.Serve(listener)
}
