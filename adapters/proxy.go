package adapters

import (
	"io"
	"log"
	"net/http"
	"os/exec"

	"github.com/moomerman/zap/multiproxy"
)

// ProxyAdapter holds the state for the application
type ProxyAdapter struct {
	Host  string
	Proxy string
	proxy *multiproxy.MultiProxy
	State Status
}

// CreateProxyAdapter creates a new proxy
func CreateProxyAdapter(host, proxy string) (Adapter, error) {
	return &ProxyAdapter{
		Host:  host,
		Proxy: proxy,
	}, nil
}

// Start starts the proxy
func (a *ProxyAdapter) Start() error {
	a.State = StatusStarting
	log.Println("[proxy]", a.Host, "starting proxy to", a.Proxy)
	a.proxy = multiproxy.NewProxy(a.Proxy, a.Host)
	a.State = StatusRunning
	return nil
}

// Status returns the status of the proxy
func (a *ProxyAdapter) Status() Status {
	return a.State
}

// ServeHTTP implements the http.Handler interface
func (a *ProxyAdapter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println("[proxy]", fullURL(r), "->", a.proxy.URL)
	a.proxy.Proxy(w, r)
}

// Stop stops the adapter
func (a *ProxyAdapter) Stop() error { return nil }

// Restart restarts the adapter
func (a *ProxyAdapter) Restart(reason error) error { return nil }

// Command doesn't do anything
func (a *ProxyAdapter) Command() *exec.Cmd { return nil }

// WriteLog doesn't do anything
func (a *ProxyAdapter) WriteLog(w io.Writer) {}
