package zap

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/moomerman/zap/cert"
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

func TestHTTPSServesHTTP2(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if err := cert.CreateCertLegacy(); err != nil {
		t.Fatal(err)
	}

	s := httpsServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(r.Proto))
	}))
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go s.ServeTLS(listener, "", "")
	defer s.Close()

	client := &http.Client{Transport: &http.Transport{
		TLSClientConfig:   &tls.Config{InsecureSkipVerify: true, ServerName: "app.test"},
		ForceAttemptHTTP2: true,
	}}
	res, err := client.Get("https://" + listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.ProtoMajor != 2 {
		t.Errorf("expected HTTP/2, got %s", res.Proto)
	}
}

func TestBootPageIsPaintedBeforeAnyOutput(t *testing.T) {
	head := fmt.Sprintf(bootPageHead, "a.test")

	// Safari waits for 200px of height and about 200 characters of visible
	// text before it paints a page that is still loading
	if !strings.Contains(head, "min-height:100vh") {
		t.Error("expected the boot page to fill the viewport")
	}
	body := head[strings.Index(head, "<body>"):]
	text := regexp.MustCompile(`<[^>]*>|\s+`).ReplaceAllString(body, "")
	if n := utf8.RuneCountInString(text); n < 200 {
		t.Errorf("expected at least 200 characters of text before any output, got %d: %q", n, text)
	}
}
