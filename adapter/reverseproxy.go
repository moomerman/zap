package adapter

import (
	"crypto/tls"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

// NewReverseProxy returns a reverse proxy to target. host is the Host header
// sent upstream; an empty host keeps the Host of each incoming request.
// Websocket upgrades are proxied by httputil.ReverseProxy itself.
func NewReverseProxy(target *url.URL, host string) *httputil.ReverseProxy {
	return &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(target)
			r.Out.Host = r.In.Host
			if host != "" {
				r.Out.Host = host
			}
			// Rewrite drops inbound X-Forwarded-* headers; keep the client's
			// X-Forwarded-For chain so SetXForwarded appends to it
			if prior, ok := r.In.Header["X-Forwarded-For"]; ok {
				r.Out.Header["X-Forwarded-For"] = prior
			}
			r.SetXForwarded()
		},
		ModifyResponse: func(resp *http.Response) error {
			resp.Header.Del("Server")
			return nil
		},
		Transport:     transport,
		FlushInterval: 250 * time.Millisecond,
	}
}

// transport is shared by every proxy so idle upstream connections are pooled.
// Upstreams are local dev servers, often with self-signed certificates, so
// their certificates aren't verified.
var transport = &http.Transport{
	Proxy: http.ProxyFromEnvironment,
	DialContext: (&net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 60 * time.Second,
	}).DialContext,
	MaxIdleConns:          100,
	IdleConnTimeout:       90 * time.Second,
	TLSHandshakeTimeout:   10 * time.Second,
	ExpectContinueTimeout: 1 * time.Second,
	TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
}
