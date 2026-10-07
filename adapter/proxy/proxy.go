package proxy

import (
	"io"
	"log"
	"net/http"
	"net/url"
	"sync"

	zadapter "github.com/moomerman/zap/adapter"
	"github.com/moomerman/zap/rproxy"
)

// New creates a new proxy
func New(host, proxy string) zadapter.Adapter {
	return &adapter{
		host:   host,
		target: proxy,
		state:  zadapter.StatusStopped,
	}
}

type adapter struct {
	host   string
	target string

	mu    sync.Mutex
	state zadapter.Status
	proxy *rproxy.ReverseProxy
}

// Start starts the proxy
func (a *adapter) Start() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	log.Println("[proxy]", a.host, "starting proxy to", a.target)
	url, err := url.Parse(a.target)
	if err != nil {
		a.state = zadapter.StatusError
		return err
	}
	proxy, err := rproxy.New(url, a.host)
	if err != nil {
		a.state = zadapter.StatusError
		return err
	}

	a.proxy = proxy
	a.state = zadapter.StatusRunning
	return nil
}

// Status returns the status of the proxy
func (a *adapter) Status() zadapter.Status {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.state
}

// Snapshot returns the current state of the proxy
func (a *adapter) Snapshot() zadapter.Snapshot {
	return zadapter.Snapshot{Name: "Proxy", Status: a.Status()}
}

// ServeHTTP implements the http.Handler interface
func (a *adapter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	proxy := a.proxy
	a.mu.Unlock()

	if proxy == nil {
		http.Error(w, "502 Bad Gateway", http.StatusBadGateway)
		return
	}

	log.Println("[proxy]", zadapter.FullURL(r), "->", proxy.URL)
	proxy.ServeHTTP(w, r)
}

// Stop stops the adapter
func (a *adapter) Stop(reason error) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.state = zadapter.StatusStopped
	return nil
}

// WriteLog doesn't do anything
func (a *adapter) WriteLog(w io.Writer) {}
