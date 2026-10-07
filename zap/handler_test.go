package zap

import (
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestProxyHandlerStartsStaticApp(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	h := &proxyHandler{manager: newTestManager(t, configs{"static.test": {Dir: dir, Key: "static"}})}

	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "http://static.test/", nil))
	if w.Code != http.StatusOK || w.Body.String() != "hello" {
		t.Errorf("expected the static file, got %d %q", w.Code, w.Body.String())
	}

	// /zap is an ordinary path now, passed through to the app
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "http://static.test/zap", nil))
	if w.Code != http.StatusNotFound {
		t.Errorf("expected the app's 404 for /zap, got %d", w.Code)
	}
}

func TestProxyHandlerUnknownHost(t *testing.T) {
	h := &proxyHandler{manager: newTestManager(t, configs{})}

	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "http://nope.test/", nil))
	if w.Code != http.StatusNotFound || !strings.Contains(w.Body.String(), "app not found") {
		t.Errorf("expected 404 app not found, got %d %q", w.Code, w.Body.String())
	}
}

func TestProxyHandlerWaitsForBoot(t *testing.T) {
	if testing.Short() {
		t.Skip("starts processes")
	}
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("needs python3")
	}
	t.Setenv("SHELL", "/bin/sh")

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("booted"), 0644); err != nil {
		t.Fatal(err)
	}
	command := `sh -c 'sleep 0.3; exec python3 -m http.server --bind 127.0.0.1 $PORT'`
	h := &proxyHandler{manager: newTestManager(t, configs{"server.test": {Dir: dir, Command: command, Port: "PORT", Scheme: "http", Key: "server"}})}

	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "http://server.test/", nil))
	if w.Code != http.StatusOK || w.Body.String() != "booted" {
		t.Errorf("expected the first request to wait for the app, got %d %q", w.Code, w.Body.String())
	}
}

func TestProxyHandlerReportsFailedBoot(t *testing.T) {
	if testing.Short() {
		t.Skip("starts processes")
	}
	t.Setenv("SHELL", "/bin/sh")

	command := `echo something broke; exit 1`
	h := &proxyHandler{manager: newTestManager(t, configs{"broken.test": {Dir: t.TempDir(), Command: command, Port: "PORT", Scheme: "http", Key: "broken"}})}

	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "http://broken.test/", nil))
	if w.Code != http.StatusBadGateway || !strings.Contains(w.Body.String(), "something broke") {
		t.Errorf("expected a 502 with the app's output, got %d %q", w.Code, w.Body.String())
	}
}

func TestProxyHandlerStreamsBootPage(t *testing.T) {
	if testing.Short() {
		t.Skip("starts processes")
	}
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("needs python3")
	}
	t.Setenv("SHELL", "/bin/sh")

	command := `sh -c 'echo compiling...; sleep 0.3; exec python3 -m http.server --bind 127.0.0.1 $PORT'`
	h := &proxyHandler{manager: newTestManager(t, configs{"server.test": {Dir: t.TempDir(), Command: command, Port: "PORT", Scheme: "http", Key: "server"}})}

	r := httptest.NewRequest("GET", "http://server.test/", nil)
	r.Header.Set("Accept", "text/html,application/xhtml+xml")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	body := w.Body.String()
	if w.Code != http.StatusServiceUnavailable || !strings.Contains(body, "compiling...") || !strings.HasSuffix(body, "location.reload()</script>") {
		t.Errorf("expected a boot page that reloads when running, got %d %q", w.Code, body)
	}

	// the reload is served by the app
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("expected the app to serve the reload, got %d", w.Code)
	}
}

func TestListenRejectsOldLaunchdSockets(t *testing.T) {
	for _, addr := range []string{"Socket", "SocketTLS"} {
		if _, err := listen(addr); err == nil || !strings.Contains(err.Error(), "-install") {
			t.Errorf("%s: got %v, want an error asking to reinstall", addr, err)
		}
	}
}

func TestLoopbackListenerRefusesOtherMachines(t *testing.T) {
	inner, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	l := &loopbackListener{inner}
	defer l.Close()

	go func() {
		if c, err := net.Dial("tcp", inner.Addr().String()); err == nil {
			c.Close()
		}
	}()
	conn, err := l.Accept()
	if err != nil {
		t.Fatal(err)
	}
	conn.Close()

	for host, want := range map[string]bool{"127.0.0.1": true, "::1": true, "localhost": true, "0.0.0.0": false, "192.168.1.5": false, "": false} {
		if got := isLoopback(host); got != want {
			t.Errorf("isLoopback(%q) = %v, want %v", host, got, want)
		}
	}
}
