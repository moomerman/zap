package zap

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/unrolled/render"
)

type contextKey string

var (
	appKey contextKey = "app"

	renderer = render.New(render.Options{
		Layout:     "layout",
		Asset:      Asset,
		AssetNames: AssetNames,
		Extensions: []string{".html"},
	})
)

type handlers struct {
	manager *Manager
}

// appView is the data passed to the HTML templates
type appView struct {
	Snapshot
	LogTail string
}

func newAppView(a *app) appView {
	return appView{Snapshot: a.snapshot(), LogTail: a.LogTail()}
}

func appFrom(r *http.Request) *app {
	return r.Context().Value(appKey).(*app)
}

func withApp(r *http.Request, a *app) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), appKey, a))
}

// ensureApp finds the app for the request host and starts it if needed
func (h *handlers) ensureApp(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		a, err := h.manager.ensure(r.Host)
		if err != nil {
			log.Println("[app]", r.Host, err)
			renderer.HTML(w, http.StatusBadGateway, "502", "App Not Found")
			return
		}
		next.ServeHTTP(w, withApp(r, a))
	}
}

// findApp finds the app for the request host without starting it, so the
// status page can keep showing why an app failed
func (h *handlers) findApp(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		a, _, err := h.manager.lookup(r.Host)
		if err != nil {
			log.Println("[app]", r.Host, err)
			renderer.HTML(w, http.StatusBadGateway, "502", "App Not Found")
			return
		}
		next.ServeHTTP(w, withApp(r, a))
	}
}

// ZAP HANDLERS

func (h *handlers) app(w http.ResponseWriter, r *http.Request) {
	a := appFrom(r)

	if a.status() == StateRunning {
		a.ServeHTTP(w, r)
		return
	}
	renderer.HTML(w, http.StatusAccepted, "app", newAppView(a))
}

func (h *handlers) status(w http.ResponseWriter, r *http.Request) {
	renderer.HTML(w, http.StatusOK, "app", newAppView(appFrom(r)))
}

func (h *handlers) log(w http.ResponseWriter, r *http.Request) {
	renderer.HTML(w, http.StatusOK, "log", newAppView(appFrom(r)))
}

func (h *handlers) restart(w http.ResponseWriter, r *http.Request) {
	a := appFrom(r)

	if err := h.manager.Restart(a.key); err != nil {
		log.Println("[app]", r.Host, "internal server error", err)
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/zap", http.StatusTemporaryRedirect)
}

func (h *handlers) logAPI(w http.ResponseWriter, r *http.Request) {
	appFrom(r).WriteLog(w)
}

// NGROK HANDLERS

func (h *handlers) ngrok(w http.ResponseWriter, r *http.Request) {
	renderer.HTML(w, http.StatusOK, "ngrok", newAppView(appFrom(r)))
}

func (h *handlers) startNgrok(w http.ResponseWriter, r *http.Request) {
	a := appFrom(r)

	if err := a.startNgrok(r.Host, 80); err != nil {
		log.Println("[app]", r.Host, "internal server error", err)
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/zap/ngrok", http.StatusTemporaryRedirect)
}

// API HANDLERS

func (h *handlers) stateAPI(w http.ResponseWriter, r *http.Request) {
	s := appFrom(r).snapshot()

	writeJSON(w, map[string]interface{}{
		"app":    s,
		"uptime": time.Since(s.Started).String(),
		"status": s.Status,
	})
}

func (h *handlers) appsAPI(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]interface{}{
		"apps": h.manager.List(),
	})
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	content, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		log.Println("[zap]", "internal server error", err)
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(content)
}
