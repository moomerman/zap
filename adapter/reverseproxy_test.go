package adapter

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// newTestProxy starts zap's reverse proxy in front of upstream. host is the
// fixed Host header to send upstream, or "" to keep the incoming one.
func newTestProxy(t *testing.T, upstream *httptest.Server, path, host string, tls bool) *httptest.Server {
	t.Helper()
	target, err := url.Parse(upstream.URL + path)
	if err != nil {
		t.Fatal(err)
	}
	proxy := NewReverseProxy(target, host)
	var front *httptest.Server
	if tls {
		front = httptest.NewTLSServer(proxy)
	} else {
		front = httptest.NewServer(proxy)
	}
	t.Cleanup(front.Close)
	return front
}

// capture records the last request an upstream received
func capture(t *testing.T) (*httptest.Server, *http.Request) {
	t.Helper()
	got := &http.Request{}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*got = *r.Clone(r.Context())
		w.Header().Set("Server", "upstream/1.0")
		w.Header().Set("X-Upstream", "yes")
		w.WriteHeader(http.StatusTeapot)
		io.WriteString(w, "hello")
	}))
	t.Cleanup(upstream.Close)
	return upstream, got
}

func TestReverseProxyRequest(t *testing.T) {
	upstream, got := capture(t)
	front := newTestProxy(t, upstream, "/base?a=1", "", false)

	req, _ := http.NewRequest("GET", front.URL+"/x/y?b=2", nil)
	req.Host = "myapp.test"
	req.Header["User-Agent"] = nil // send no User-Agent at all
	req.Header.Set("X-Forwarded-For", "10.0.0.1")
	req.Header.Set("X-Forwarded-Proto", "https") // a client can't spoof this
	req.Header.Set("X-Custom", "kept")
	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	if resp.StatusCode != http.StatusTeapot || string(body) != "hello" {
		t.Errorf("response = %d %q", resp.StatusCode, body)
	}
	if v := resp.Header.Get("Server"); v != "" {
		t.Errorf("Server header = %q, want it stripped", v)
	}
	if v := resp.Header.Get("X-Upstream"); v != "yes" {
		t.Errorf("X-Upstream = %q, want upstream headers passed back", v)
	}

	checks := map[string][2]string{
		"path":              {got.URL.Path, "/base/x/y"},
		"query":             {got.URL.RawQuery, "a=1&b=2"},
		"host":              {got.Host, "myapp.test"},
		"user-agent":        {got.Header.Get("User-Agent"), ""},
		"x-forwarded-proto": {got.Header.Get("X-Forwarded-Proto"), "http"},
		"x-forwarded-for":   {got.Header.Get("X-Forwarded-For"), "10.0.0.1, 127.0.0.1"},
		"x-custom":          {got.Header.Get("X-Custom"), "kept"},
	}
	for name, c := range checks {
		if c[0] != c[1] {
			t.Errorf("%s = %q, want %q", name, c[0], c[1])
		}
	}
	if _, ok := got.Header["User-Agent"]; ok {
		t.Errorf("User-Agent was sent upstream: %q", got.Header["User-Agent"])
	}
}

func TestReverseProxyUserAgentPassedThrough(t *testing.T) {
	upstream, got := capture(t)
	front := newTestProxy(t, upstream, "", "", false)

	req, _ := http.NewRequest("GET", front.URL, nil)
	req.Header.Set("User-Agent", "browser/1.0")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if v := got.Header.Get("User-Agent"); v != "browser/1.0" {
		t.Errorf("User-Agent = %q", v)
	}
}

func TestReverseProxyFixedHost(t *testing.T) {
	upstream, got := capture(t)
	front := newTestProxy(t, upstream, "", "configured.test", false)

	req, _ := http.NewRequest("GET", front.URL+"/", nil)
	req.Host = "sub.configured.test"
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if got.Host != "configured.test" {
		t.Errorf("host = %q, want configured.test", got.Host)
	}
}

func TestReverseProxyHTTPS(t *testing.T) {
	// an https upstream with a self-signed certificate, behind an https front
	upstream := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, r.Header.Get("X-Forwarded-Proto"))
	}))
	defer upstream.Close()
	front := newTestProxy(t, upstream, "", "", true)

	resp, err := front.Client().Get(front.URL)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || string(body) != "https" {
		t.Errorf("response = %d %q, want 200 \"https\"", resp.StatusCode, body)
	}
}

func TestReverseProxyUpstreamDown(t *testing.T) {
	upstream := httptest.NewServer(http.NotFoundHandler())
	front := newTestProxy(t, upstream, "", "", false)
	upstream.Close()

	resp, err := http.Get(front.URL)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadGateway {
		t.Errorf("status = %d, want 502", resp.StatusCode)
	}
}

// A streamed response must reach the client before the upstream finishes.
func TestReverseProxyStreaming(t *testing.T) {
	release := make(chan struct{})
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "data: one\n\n")
		w.(http.Flusher).Flush()
		<-release
	}))
	defer upstream.Close()
	defer close(release)
	front := newTestProxy(t, upstream, "", "", false)

	resp, err := http.Get(front.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	line := make(chan string, 1)
	go func() {
		l, _ := bufio.NewReader(resp.Body).ReadString('\n')
		line <- l
	}()
	select {
	case l := <-line:
		if l != "data: one\n" {
			t.Errorf("got %q", l)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("streamed event was not flushed to the client")
	}
}

// wsUpstream accepts a websocket-style upgrade and echoes bytes back, after
// reporting the headers it received.
func wsUpstream(t *testing.T) *httptest.Server {
	t.Helper()
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
			http.Error(w, "expected upgrade", http.StatusBadRequest)
			return
		}
		conn, rw, err := w.(http.Hijacker).Hijack()
		if err != nil {
			return
		}
		defer conn.Close()
		fmt.Fprintf(rw, "HTTP/1.1 101 Switching Protocols\r\n"+
			"Upgrade: websocket\r\nConnection: Upgrade\r\n"+
			"Sec-WebSocket-Accept: accept-for-%s\r\n"+
			"X-Seen-Host: %s\r\nX-Seen-Proto: %s\r\nX-Seen-Protocol: %s\r\nX-Seen-Cookie: %s\r\n\r\n",
			r.Header.Get("Sec-WebSocket-Key"), r.Host, r.Header.Get("X-Forwarded-Proto"),
			r.Header.Get("Sec-WebSocket-Protocol"), r.Header.Get("Cookie"))
		rw.Flush()
		io.Copy(conn, rw)
	}))
	t.Cleanup(upstream.Close)
	return upstream
}

func TestReverseProxyWebsocket(t *testing.T) {
	for _, useTLS := range []bool{false, true} {
		t.Run(fmt.Sprintf("tls=%v", useTLS), func(t *testing.T) {
			front := newTestProxy(t, wsUpstream(t), "", "", useTLS)
			addr := strings.TrimPrefix(strings.TrimPrefix(front.URL, "http://"), "https://")

			var conn net.Conn
			var err error
			proto := "http"
			if useTLS {
				conn, err = tls.Dial("tcp", addr, &tls.Config{InsecureSkipVerify: true, NextProtos: []string{"http/1.1"}})
				proto = "https"
			} else {
				conn, err = net.Dial("tcp", addr)
			}
			if err != nil {
				t.Fatal(err)
			}
			defer conn.Close()
			conn.SetDeadline(time.Now().Add(5 * time.Second))

			fmt.Fprint(conn, "GET /socket HTTP/1.1\r\nHost: myapp.test\r\n"+
				"Upgrade: websocket\r\nConnection: Upgrade\r\n"+
				"Sec-WebSocket-Key: abc123\r\nSec-WebSocket-Version: 13\r\n"+
				"Sec-WebSocket-Protocol: phoenix\r\nCookie: session=s1\r\n\r\n")

			br := bufio.NewReader(conn)
			resp, err := http.ReadResponse(br, nil)
			if err != nil {
				t.Fatal(err)
			}
			if resp.StatusCode != http.StatusSwitchingProtocols {
				t.Fatalf("status = %d, want 101", resp.StatusCode)
			}
			for header, want := range map[string]string{
				"Upgrade":              "websocket",
				"Sec-Websocket-Accept": "accept-for-abc123",
				"X-Seen-Host":          "myapp.test",
				"X-Seen-Proto":         proto,
				"X-Seen-Protocol":      "phoenix",
				"X-Seen-Cookie":        "session=s1",
			} {
				if v := resp.Header.Get(header); v != want {
					t.Errorf("%s = %q, want %q", header, v, want)
				}
			}

			for _, msg := range []string{"ping\n", "second frame\n"} {
				if _, err := io.WriteString(conn, msg); err != nil {
					t.Fatal(err)
				}
				echo, err := br.ReadString('\n')
				if err != nil {
					t.Fatal(err)
				}
				if echo != msg {
					t.Errorf("echo = %q, want %q", echo, msg)
				}
			}
		})
	}
}
