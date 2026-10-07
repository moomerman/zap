package zap

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"
)

// startTimeout is how long a request waits for its app to boot
const startTimeout = 60 * time.Second

// proxyHandler serves requests for app hosts, starting the app if needed
type proxyHandler struct {
	manager *Manager
}

func (h *proxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a, err := h.manager.ensure(r.Host)
	if err != nil {
		log.Println("[app]", r.Host, err)
		status := http.StatusBadGateway
		if errors.Is(err, ErrNotFound) {
			status = http.StatusNotFound
		}
		writeError(w, status, err.Error(), "")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), startTimeout)
	state := a.waitStarted(ctx)
	cancel()

	switch state {
	case StateRunning:
		a.ServeHTTP(w, r)
	case StateStarting:
		writeError(w, http.StatusGatewayTimeout, "app is still starting, try again shortly", a.LogTail())
	default:
		s := a.snapshot()
		msg := fmt.Sprintf("app is %s", s.Status)
		if s.Error != "" {
			msg += ": " + s.Error
		}
		writeError(w, http.StatusBadGateway, msg, a.LogTail())
	}
}

// writeError writes a plain text error page with the app's recent output
func writeError(w http.ResponseWriter, status int, msg, logTail string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	fmt.Fprintf(w, "⚡ zap: %d %s\n\n%s\n", status, http.StatusText(status), msg)
	if logTail != "" {
		fmt.Fprintf(w, "\n%s", logTail)
	}
}
