package adapters

import (
	"fmt"
	"io"
	"net/http"
	"os/exec"

	"github.com/moomerman/phx-dev/multiproxy"
)

// ProxyAdapter holds the state for the application
type ProxyAdapter struct {
	Host  string
	Port  string
	proxy *multiproxy.MultiProxy
}

// CreateProxyAdapter creates a new proxy
func CreateProxyAdapter(host, port string) (Adapter, error) {
	return &ProxyAdapter{
		Host: host,
		Port: port,
	}, nil
}

// Start starts the proxy
func (d *ProxyAdapter) Start() error {
	// TODO: read proxy host/port from file
	addr := "http://127.0.0.1:" + d.Port
	fmt.Println("[proxy]", d.Host, "starting proxy to", addr)
	d.proxy = multiproxy.NewProxy(addr, d.Host)
	return nil
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
