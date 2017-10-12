package proxy

import (
	"io"
	"log"
	"net/http"
	"os/exec"

	zadapter "github.com/moomerman/zap/adapter"
	"github.com/moomerman/zap/proxy"
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
	proxy   *proxy.MultiProxy
	State   zadapter.Status
	BootLog string
}

// Start starts the proxy
func (a *adapter) Start() error {
	a.State = zadapter.StatusStarting
	log.Println("[proxy]", a.Host, "starting proxy to", a.Proxy)
	proxy, err := proxy.NewProxy(a.Proxy, a.Host)
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
	a.proxy.Proxy(w, r)
}

// Stop stops the adapter
func (a *adapter) Stop(reason error) error { return nil }

// Command doesn't do anything
func (a *adapter) Command() *exec.Cmd { return nil }

// WriteLog doesn't do anything
func (a *adapter) WriteLog(w io.Writer) {}
