package adapter

import (
	"fmt"
	"io"
	"net/http"
	"os/exec"

	"github.com/moomerman/phx-dev/multiproxy"
)

// ProxyAdapter holds the state for a simple proxy
type ProxyAdapter struct {
	Host  string
	Dir   string
	Port  string
	proxy *multiproxy.MultiProxy
}

// CreateProxyAdapter creates a new proxy
func CreateProxyAdapter(host, dir, port string) (Adapter, error) {
	return &ProxyAdapter{
		Host: host,
		Dir:  dir,
		Port: port,
	}, nil
}

func (d *ProxyAdapter) Stop() error          { return nil }
func (d *ProxyAdapter) Command() *exec.Cmd   { return nil }
func (d *ProxyAdapter) WriteLog(w io.Writer) {}

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
