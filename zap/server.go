package zap

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/moomerman/zap/cert"
	"github.com/puma/puma-dev/dev/launch"
	"github.com/unrolled/render"
	"golang.org/x/net/http2"
)

// Server holds the state for the HTTP and HTTPS servers
type Server struct {
	http  *http.Server
	https *http.Server
}

// NewServer starts the HTTP and HTTPS proxy servers
func NewServer() *Server {
	httpsMux := http.NewServeMux()
	// TODO: don't handle these requests unless localhost request (eg. not via ngrok)
	httpsMux.HandleFunc("/zap/api/log", findAppHandler(logAPIHandler()))
	httpsMux.HandleFunc("/zap/api/state", findAppHandler(stateAPIHandler()))
	httpsMux.HandleFunc("/zap/api/apps", appsAPIHandler())
	httpsMux.HandleFunc("/zap/log", findAppHandler(logHandler()))
	httpsMux.HandleFunc("/zap/restart", findAppHandler(restartHandler()))
	httpsMux.HandleFunc("/zap", findAppHandler(statusHandler()))
	httpsMux.HandleFunc("/", findAppHandler(appHandler()))

	httpMux := http.NewServeMux()
	httpMux.HandleFunc("/", findAppHandler(appHandler()))

	http := startHTTP(httpMux)
	https := startHTTPS(httpsMux)

	return &Server{
		http:  http,
		https: https,
	}
}

var renderer = render.New(render.Options{
	Layout:     "layout",
	Asset:      Asset,
	AssetNames: AssetNames,
	Extensions: []string{".html"},
})

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
	if bind == "Socket" {
		listeners, err := launch.SocketListeners(bind)
		if err != nil {
			log.Fatal("unable to get launchd socket listener", err)
		}
		s.http.Serve(listeners[0])
	} else {
		listener, err := net.Listen("tcp", bind)
		if err != nil {
			log.Fatal("unable to create listener", err)
		}
		s.http.Serve(listener)
	}
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
	return &http.Server{
		Handler: handler,
	}
}

type contextKey string

var appKey contextKey = "app"

func findAppHandler(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		app, err := findAppForHost(r.Host)
		if err != nil {
			renderer.HTML(w, http.StatusBadGateway, "502", "App Not Found")
			return
		}
		ctx := context.WithValue(r.Context(), appKey, app)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func appHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		app := r.Context().Value(appKey).(*app)

		switch app.Status() {
		case "running":
			app.ServeHTTP(w, r)
		default:
			renderer.HTML(w, http.StatusAccepted, "app", app)
		}
	}
}

func statusHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		app := r.Context().Value(appKey).(*app)
		renderer.HTML(w, http.StatusOK, "app", app)
	}
}

func logHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		app := r.Context().Value(appKey).(*app)
		renderer.HTML(w, http.StatusOK, "log", app)
	}
}

func restartHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		app := r.Context().Value(appKey).(*app)

		if err := app.Restart(); err != nil {
			log.Println("[app]", app.Config.Host, "internal server error", err)
			http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		}

		http.Redirect(w, r, "/zap", http.StatusTemporaryRedirect)
	}
}

func logAPIHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		app := r.Context().Value(appKey).(*app)

		app.WriteLog(w)
	}
}

func stateAPIHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		app := r.Context().Value(appKey).(*app)

		content, _ := json.MarshalIndent(map[string]interface{}{
			"app":    app,
			"uptime": time.Since(app.Started).String(),
			"status": app.Status(),
		}, "", "  ")

		w.Header().Set("Content-Type", "application/json")
		w.Write(content)
	}
}

func appsAPIHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {

		content, err := json.MarshalIndent(map[string]interface{}{
			"apps": apps,
		}, "", "  ")
		if err != nil {
			log.Println("[app] internal server error", err)
			http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(content)
	}
}
