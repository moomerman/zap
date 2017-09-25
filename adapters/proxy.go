package adapters

import (
	"fmt"
	"io"
	"net/http"
	"os/exec"

	"github.com/moomerman/zap/multiproxy"
)

// ProxyAdapter holds the state for the application
type ProxyAdapter struct {
	Host  string
	Proxy string
	proxy *multiproxy.MultiProxy
	state Status
}

// CreateProxyAdapter creates a new proxy
func CreateProxyAdapter(host, proxy string) (Adapter, error) {
	return &ProxyAdapter{
		Host:  host,
		Proxy: proxy,
	}, nil
}

// Start starts the proxy
func (d *ProxyAdapter) Start() error {
	d.state = StatusStarting
	fmt.Println("[proxy]", d.Host, "starting proxy to", d.Proxy)
	d.proxy = multiproxy.NewProxy(d.Proxy, d.Host)
	d.state = StatusRunning
	return nil
}

// Status returns the status of the proxy
func (d *ProxyAdapter) Status() Status {
	return d.state
}

// ServeHTTP implements the http.Handler interface
func (d *ProxyAdapter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Println("[proxy]", fullURL(r), "->", d.proxy.URL)
	d.proxy.Proxy(w, r)
}

// Stop doesn't do anything
func (d *ProxyAdapter) Stop() error { return nil }

// Command doesn't do anything
func (d *ProxyAdapter) Command() *exec.Cmd { return nil }

// WriteLog doesn't do anything
func (d *ProxyAdapter) WriteLog(w io.Writer) {}
