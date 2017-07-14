package dev

import (
	"fmt"
	"io"
	"net/http"
	"os/exec"

	"github.com/moomerman/phx-dev/multiproxy"
)

type ProxyDriver struct {
	Host  string
	Dir   string
	Port  string
	proxy *multiproxy.MultiProxy
}

func CreateProxyDriver(host, dir, port string) (Driver, error) {
	return &ProxyDriver{
		Host: host,
		Dir:  dir,
		Port: port,
	}, nil
}

func (d *ProxyDriver) Stop() error          { return nil }
func (d *ProxyDriver) Command() *exec.Cmd   { return nil }
func (d *ProxyDriver) WriteLog(w io.Writer) {}

func (d *ProxyDriver) Start() error {
	addr := "http://127.0.0.1:" + d.Port
	fmt.Println("[proxy]", d.Host, "starting proxy to", addr)
	d.proxy = multiproxy.NewProxy(addr, d.Host)
	return nil
}

func (d *ProxyDriver) Serve(w http.ResponseWriter, r *http.Request) {
	fmt.Println("[proxy]", fullURL(r), "->", d.proxy.URL)
	d.proxy.Proxy(w, r)
}
