package control

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/moomerman/zap/zap"
)

// Manager is the part of zap.Manager the control API uses
type Manager interface {
	List() []zap.Snapshot
	Get(host string) (zap.Snapshot, error)
	Start(key string) error
	Stop(key string) error
	Restart(key string) error
	WriteLog(key string, w io.Writer) error
	Subscribe() (<-chan zap.Event, func())
}

// Server serves the control API
type Server struct {
	manager Manager
	http    *http.Server
	done    chan struct{} // closed on shutdown to end event streams

	closeOnce sync.Once
}

// NewServer returns a control API server for m
func NewServer(m Manager) *Server {
	s := &Server{manager: m, done: make(chan struct{})}
	s.http = &http.Server{Handler: s.Handler()}
	return s
}

// Handler returns the control API's routes
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/apps", s.list)
	mux.HandleFunc("GET /v1/apps/{host}", s.get)
	mux.HandleFunc("POST /v1/apps/{host}/start", s.action(Manager.Start))
	mux.HandleFunc("POST /v1/apps/{host}/stop", s.action(Manager.Stop))
	mux.HandleFunc("POST /v1/apps/{host}/restart", s.action(Manager.Restart))
	mux.HandleFunc("GET /v1/apps/{host}/log", s.log)
	mux.HandleFunc("GET /v1/events", s.events)
	return mux
}

// ListenAndServe serves the API on a unix socket at path, readable only by
// the current user, until Shutdown is called
func (s *Server) ListenAndServe(path string) error {
	l, err := listen(path)
	if err != nil {
		return err
	}
	defer os.Remove(path)

	log.Println("[zap] control API listening at", path)
	return s.http.Serve(l)
}

// Shutdown stops the server. Open event streams are closed.
func (s *Server) Shutdown(ctx context.Context) error {
	s.closeOnce.Do(func() { close(s.done) })
	return s.http.Shutdown(ctx)
}

func listen(path string) (net.Listener, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}

	// a socket left behind by a zapd that didn't exit cleanly is removed, but
	// not one that another zapd is still serving
	if _, err := os.Stat(path); err == nil {
		if conn, err := net.DialTimeout("unix", path, time.Second); err == nil {
			conn.Close()
			return nil, fmt.Errorf("%s is in use, is zapd already running?", path)
		}
		os.Remove(path)
	}

	l, err := net.Listen("unix", path)
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(path, 0600); err != nil {
		l.Close()
		return nil, err
	}
	return l, nil
}

func (s *Server) list(w http.ResponseWriter, r *http.Request) {
	snapshots := s.manager.List()
	apps := make([]App, 0, len(snapshots))
	for _, snapshot := range snapshots {
		apps = append(apps, newApp(snapshot))
	}
	writeJSON(w, http.StatusOK, apps)
}

func (s *Server) get(w http.ResponseWriter, r *http.Request) {
	snapshot, err := s.manager.Get(r.PathValue("host"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, newApp(snapshot))
}

// action runs f on the app for the request's host and returns its new state
func (s *Server) action(f func(Manager, string) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		host := r.PathValue("host")
		snapshot, err := s.manager.Get(host)
		if err != nil {
			writeError(w, err)
			return
		}
		if err := f(s.manager, snapshot.Key); err != nil {
			writeError(w, err)
			return
		}
		s.get(w, r)
	}
}

func (s *Server) log(w http.ResponseWriter, r *http.Request) {
	snapshot, err := s.manager.Get(r.PathValue("host"))
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if err := s.manager.WriteLog(snapshot.Key, w); err != nil {
		writeError(w, err)
	}
}

// events streams app events as server-sent events. With ?host= only events
// for that host's app are sent, including other hosts that share its process.
func (s *Server) events(w http.ResponseWriter, r *http.Request) {
	var key string
	if host := r.URL.Query().Get("host"); host != "" {
		snapshot, err := s.manager.Get(host)
		if err != nil {
			writeError(w, err)
			return
		}
		key = snapshot.Key
	}

	events, unsubscribe := s.manager.Subscribe()
	defer unsubscribe()

	rc := http.NewResponseController(w)
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	rc.Flush()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-s.done:
			return
		case e, ok := <-events:
			if !ok {
				return
			}
			if key != "" && e.Key != key {
				continue
			}
			data, err := json.Marshal(newEvent(e))
			if err != nil {
				continue
			}
			if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", e.Type, data); err != nil {
				return
			}
			if err := rc.Flush(); err != nil {
				return
			}
		}
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.Encode(v)
}

func writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, zap.ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, zap.ErrShutdown):
		status = http.StatusServiceUnavailable
	}
	writeJSON(w, status, errorResponse{Error: err.Error()})
}
