package zap

import (
	"context"
	"crypto/tls"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/moomerman/zap/cert"
	"github.com/puma/puma-dev/dev/launch"
	"golang.org/x/net/http2"
)

// Server holds the state for the HTTP and HTTPS servers
type Server struct {
	http  *http.Server
	https *http.Server
}

// NewServer creates the HTTP and HTTPS servers
func NewServer() *Server {
	return &Server{
		http:  createHTTPServer(),
		https: createHTTPSServer(),
	}
}

// ServeTLS starts the HTTPS server
func (s *Server) ServeTLS(bind string) {
	if bind == "SocketTLS" {
		listeners, err := launch.SocketListeners(bind)
		if err != nil {
			log.Fatal("unable to get launchd socket listener", err)
		}
		log.Println("[zap] https listening at", "SocketTLS (launchd)")
		s.https.Serve(tls.NewListener(listeners[0], s.https.TLSConfig))
	} else {
		listener, err := tls.Listen("tcp", bind, s.https.TLSConfig)
		if err != nil {
			log.Fatal("unable to create tls listener", err)
		}
		log.Println("[zap] https listening at", bind)
		s.https.Serve(listener)
	}
}

// Serve starts the HTTP server
func (s *Server) Serve(bind string) {
	if bind == "Socket" {
		listeners, err := launch.SocketListeners(bind)
		if err != nil {
			log.Fatal("unable to get launchd socket listener", err)
		}
		log.Println("[zap] http listening at", "Socket (launchd)")
		s.http.Serve(listeners[0])
	} else {
		listener, err := net.Listen("tcp", bind)
		if err != nil {
			log.Fatal("unable to create listener", err)
		}
		log.Println("[zap] http listening at", bind)
		s.http.Serve(listener)
	}
}

// Stop gracefully stops the HTTP and HTTPS servers
func (s *Server) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	s.http.Shutdown(ctx)
	s.https.Shutdown(ctx)
}

func createHTTPSServer() *http.Server {
	mux := http.NewServeMux()
	// TODO: don't handle these requests unless localhost request (eg. not via ngrok)
	// Maybe have a zapHandler that checks for localhost and then delegates requests
	mux.HandleFunc("/zap/api/log", findAppHandler(logAPIHandler()))
	mux.HandleFunc("/zap/api/state", findAppHandler(stateAPIHandler()))
	mux.HandleFunc("/zap/api/apps", appsAPIHandler())
	mux.HandleFunc("/zap/ngrok/start", findAppHandler(startNgrokHandler()))
	mux.HandleFunc("/zap/ngrok", findAppHandler(ngrokHandler()))
	mux.HandleFunc("/zap/log", findAppHandler(logHandler()))
	mux.HandleFunc("/zap/restart", findAppHandler(restartHandler()))
	mux.HandleFunc("/zap", findAppHandler(statusHandler()))
	mux.HandleFunc("/", findAppHandler(appHandler()))

	cache, err := cert.NewCache()
	if err != nil {
		log.Fatal("unable to create new cert cache", err)
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

func createHTTPServer() *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/", findAppHandler(appHandler()))

	return &http.Server{
		Handler: mux,
	}
}
