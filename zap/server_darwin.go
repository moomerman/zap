package zap

import (
	"crypto/tls"
	"log"
	"net"

	"github.com/puma/puma-dev/dev/launch"
)

func (s *Server) serveHTTP() error {
	var listener net.Listener
	var err error

	if s.HTTPAddr == "Socket" {
		listener = getSocketListener(s.HTTPAddr)
	} else {
		listener, err = net.Listen("tcp", s.HTTPAddr)
		if err != nil {
			log.Fatal("[zap] unable to create listener", err)
		}
	}

	log.Println("[zap] http listening at", listener.Addr())
	return s.http.Serve(listener)
}

func (s *Server) serveHTTPS() error {
	var listener net.Listener
	var err error

	if s.HTTPSAddr == "SocketTLS" {
		listener = tls.NewListener(getSocketListener(s.HTTPSAddr), s.https.TLSConfig)
	} else {
		listener, err = tls.Listen("tcp", s.HTTPSAddr, s.https.TLSConfig)
		if err != nil {
			log.Fatal("[zap] unable to create tls listener", err)
		}
	}

	log.Println("[zap] https listening at", listener.Addr())
	return s.https.Serve(listener)
}

func getSocketListener(socket string) net.Listener {
	listeners, err := launch.SocketListeners(socket)
	if err != nil {
		log.Fatal("[zap] unable to get launchd socket listener", err)
	}
	return listeners[0]
}
