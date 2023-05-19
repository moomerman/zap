package proxy

import (
	"io"
	"log"
	"net/http"
	"net/url"
	"os/exec"

	zadapter "github.com/moomerman/zap/adapter"
	"github.com/moomerman/zap/rproxy"
)

// New creates a new proxy
func New(host, proxy string) (zadapter.Adapter, error) {
	return &adapter{
		Name:  "Proxy",
		Host:  host,
		Proxy: proxy,
	}, nil
}

type adapter struct {
	Name    string
	Host    string
	Proxy   string
	proxy   *rproxy.ReverseProxy
	State   zadapter.Status
	BootLog string
}

// Start starts the proxy
func (a *adapter) Start() error {
	a.State = zadapter.StatusStarting
	log.Println("[proxy]", a.Host, "starting proxy to", a.Proxy)
	url, err := url.Parse(a.Proxy)
	if err != nil {
		return err
	}
	proxy, err := rproxy.New(url, a.Host)
	if err != nil {
		return err
	}

	a.proxy = proxy
	a.State = zadapter.StatusRunning
	return nil
}

// Status returns the status of the proxy
func (a *adapter) Status() zadapter.Status {
	return a.State
}

// ServeHTTP implements the http.Handler interface
func (a *adapter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println("[proxy]", zadapter.FullURL(r), "->", a.proxy.URL)
	a.proxy.ServeHTTP(w, r)
}

// Stop stops the adapter
func (a *adapter) Stop(reason error) error { return nil }

// Command doesn't do anything
func (a *adapter) Command() *exec.Cmd { return nil }

// WriteLog doesn't do anything
func (a *adapter) WriteLog(w io.Writer) {}
