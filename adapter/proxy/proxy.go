package proxy

import (
	"io"
	"log"
	"net/http"
	"os/exec"

	"github.com/moomerman/zap/adapter"
	"github.com/moomerman/zap/proxy"
)

// Adapter holds the state for the application
type Adapter struct {
	Name    string
	Host    string
	Proxy   string
	proxy   *proxy.MultiProxy
	State   adapter.Status
	BootLog string
}

// New creates a new proxy
func New(host, proxy string) (adapter.Adapter, error) {
	return &Adapter{
		Name:  "Proxy",
		Host:  host,
		Proxy: proxy,
	}, nil
}

// Start starts the proxy
func (a *Adapter) Start() error {
	a.State = adapter.StatusStarting
	log.Println("[proxy]", a.Host, "starting proxy to", a.Proxy)
	proxy, err := proxy.NewProxy(a.Proxy, a.Host)
	if err != nil {
		return err
	}

	a.proxy = proxy
	a.State = adapter.StatusRunning
	return nil
}

// Status returns the status of the proxy
func (a *Adapter) Status() adapter.Status {
	return a.State
}

// ServeHTTP implements the http.Handler interface
func (a *Adapter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println("[proxy]", adapter.FullURL(r), "->", a.proxy.URL)
	a.proxy.Proxy(w, r)
}

// Stop stops the adapter
func (a *Adapter) Stop(reason error) error { return nil }

// Command doesn't do anything
func (a *Adapter) Command() *exec.Cmd { return nil }

// WriteLog doesn't do anything
func (a *Adapter) WriteLog(w io.Writer) {}
