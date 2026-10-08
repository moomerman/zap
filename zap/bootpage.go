package zap

import (
	"context"
	"fmt"
	"html"
	"io"
	"net/http"
	"strings"
)

// wantsHTML reports whether r looks like a browser navigating to a page
func wantsHTML(r *http.Request) bool {
	return r.Method == http.MethodGet && strings.Contains(r.Header.Get("Accept"), "text/html")
}

// bootPageHead opens the boot page. Safari (WebKit) keeps showing the
// previous page during a navigation until the new one is "visually non-empty":
// at least 200px tall with 200 or so characters of text, or the response has
// finished. A quiet boot never gets there, so the page is a full viewport high
// and opens with enough text to be painted straight away.
const bootPageHead = `<!doctype html>
<html><head><meta charset="utf-8"><title>⚡ %[1]s is starting</title>
<style>
html,body{min-height:100vh}
body{margin:0;font:13px/1.4 ui-monospace,Menlo,monospace;background:#1d1f21;color:#c5c8c6}
h1{position:sticky;top:0;margin:0;padding:12px 16px;font-size:14px;background:#282a2e;color:#f0c674}
p{margin:0;padding:12px 16px 0;color:#969896}
pre{margin:0;padding:12px 16px;white-space:pre-wrap}
.err{color:#cc6666}
</style>
<script>setInterval(function(){window.scrollTo(0,document.body.scrollHeight)},200)</script>
</head><body><h1>⚡ %[1]s is starting…</h1>
<p>zap is starting this app. Its output appears below as it boots, and this page
reloads by itself as soon as the app is ready. If the app fails to start, the
error is shown at the end of the output. To follow it from a terminal instead,
run: zap logs -f %[1]s</p><pre>`

// serveBootPage streams an app's output to the browser while it boots and
// reloads the page once it's running. The response is one long request, so
// app hosts need no extra routes. It returns false, having written nothing,
// if the app isn't starting.
func (h *proxyHandler) serveBootPage(w http.ResponseWriter, r *http.Request, a *app) bool {
	// subscribe before checking the state so no transition is missed
	events, unsubscribe := h.manager.Subscribe()
	defer unsubscribe()

	if a.status() != StateStarting {
		return false
	}

	ctx, cancel := context.WithTimeout(r.Context(), startTimeout)
	defer cancel()

	rc := http.NewResponseController(w)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusServiceUnavailable)

	fmt.Fprintf(w, bootPageHead, html.EscapeString(r.Host))
	io.WriteString(w, html.EscapeString(a.LogTail()))
	rc.Flush()

	for {
		select {
		case <-ctx.Done():
			io.WriteString(w, `</pre><p class="err">Still starting, reload to keep waiting.</p>`)
			return true
		case e, ok := <-events:
			if !ok {
				return true
			}
			if e.Key != a.key {
				continue
			}
			switch {
			case e.Type == EventLog:
				io.WriteString(w, html.EscapeString(e.Line))
			case e.Status == StateRunning:
				io.WriteString(w, `</pre><script>location.reload()</script>`)
				return true
			case e.Status == StateError || e.Status == StateStopping || e.Status == StateStopped:
				msg := "app is " + string(e.Status)
				if e.Error != "" {
					msg += ": " + e.Error
				}
				fmt.Fprintf(w, `</pre><p class="err">%s</p>`, html.EscapeString(msg))
				return true
			}
			rc.Flush()
		}
	}
}
