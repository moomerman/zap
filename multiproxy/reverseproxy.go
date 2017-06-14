package multiproxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/yhat/wsutil"
)

type MultiProxy struct {
	URL      *url.URL
	Hostname string
	proxy    *httputil.ReverseProxy
	wsproxy  *wsutil.ReverseProxy
}

func NewProxy(targetUrl string, hostname string) *MultiProxy {
	target, err := url.Parse(targetUrl)
	if err != nil {
		panic(err) // TODO: remove this!
	}

	targetQuery := target.RawQuery

	director := func(req *http.Request) {
		if hostname != "" {
			req.Host = hostname
		}
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.URL.Path = singleJoiningSlash(target.Path, req.URL.Path)
		if targetQuery == "" || req.URL.RawQuery == "" {
			req.URL.RawQuery = targetQuery + req.URL.RawQuery
		} else {
			req.URL.RawQuery = targetQuery + "&" + req.URL.RawQuery
		}
		if _, ok := req.Header["User-Agent"]; !ok {
			// explicitly disable User-Agent so it's not set to default value
			req.Header.Set("User-Agent", "")
		}
	}

	transport := &myTransport{
		stripHeaders: []string{"Server"},
	}

	proxy := &httputil.ReverseProxy{
		Transport:     transport,
		Director:      director,
		FlushInterval: 250 * time.Millisecond,
	}

	wsproxy := &wsutil.ReverseProxy{
		Director: director,
	}

	return &MultiProxy{
		URL:      target,
		Hostname: hostname,
		proxy:    proxy,
		wsproxy:  wsproxy,
	}
}

func (p *MultiProxy) Proxy(w http.ResponseWriter, r *http.Request) {
	if r.TLS != nil {
		r.Header.Set("x-forwarded-proto", "https")
	} else {
		r.Header.Set("x-forwarded-proto", "http")
	}

	if r.Header.Get("Connection") == "Upgrade" && r.Header.Get("Upgrade") == "websocket" {
		p.wsproxy.ServeHTTP(w, r)
	} else {
		p.proxy.ServeHTTP(w, r)
	}
}

type myTransport struct {
	*http.Transport
	stripHeaders []string
}

func (t *myTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	for _, hdr := range t.stripHeaders {
		resp.Header.Del(hdr)
	}
	return resp, nil
}

// https://golang.org/src/net/http/httputil/reverseproxy.go
func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}
