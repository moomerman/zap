package dev

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/moomerman/zap/adapters"
	"github.com/puma/puma-dev/homedir"
	"github.com/vektra/errors"
)

const appsPath = "~/.zap"

var apps map[string]*App
var lock sync.Mutex

// App holds the state of a running Application
type App struct {
	Host     string
	Link     string
	Dir      string
	LastUsed time.Time

	adapter adapters.Adapter
}

// NewApp creates a new App for the given host
func NewApp(host string) (*App, error) {
	path := homedir.MustExpand(appsPath) + "/" + host
	stat, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	var dir string
	var adapter adapters.Adapter

	if stat.IsDir() {
		dir, err = os.Readlink(path)
		if err != nil {
			return nil, err
		}

		adapter, err = getAdapter(host, dir)
		if err != nil {
			return nil, errors.Context(err, "could not determine adapter")
		}
	} else {
		fmt.Println("[app]", host, "using the proxy adapter")
		// TODO: read the proxy host/port from the file
		// see https://github.com/puma/puma-dev/blob/master/dev/app.go#L473
		adapter, err = adapters.CreateProxyAdapter(host, "80")
		if err != nil {
			return nil, errors.Context(err, "unable to create proxy adapter")
		}
	}

	return &App{
		Host:    host,
		Link:    path,
		Dir:     dir,
		adapter: adapter,
	}, nil
}

// Start starts an application and monitors activity
func (a *App) Start() error {
	err := a.adapter.Start()
	if err != nil {
		return err
	}

	go a.idleMonitor()
	return nil
}

// Stop stops an application
func (a *App) Stop(reason string, e error) error {
	fmt.Printf("! Stopping '%s' %s %s\n", a.Host, reason, e)
	lock.Lock()
	delete(apps, a.Host)
	lock.Unlock()
	return a.adapter.Stop()
}

func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.LastUsed = time.Now()
	a.adapter.ServeHTTP(w, r)
}

// WriteLog writes out the application log to the given writer
func (a *App) WriteLog(w io.Writer) {
	a.adapter.WriteLog(w)
}

func getAdapter(host, dir string) (adapters.Adapter, error) {
	_, err := os.Stat(path.Join(dir, "mix.exs"))
	if err == nil {
		fmt.Println("[app]", host, "using the phoenix adapter (found mix.exs)")
		return adapters.CreatePhoenixAdapter(host, dir)
	}

	_, err = os.Stat(path.Join(dir, "Gemfile"))
	if err == nil {
		fmt.Println("[app]", host, "using the rails adapter (found Gemfile)")
		return adapters.CreateRailsAdapter(host, dir)
	}

	_, err = os.Stat(path.Join(dir, ".buffalo.dev.yml"))
	if err == nil {
		fmt.Println("[app]", host, "using the buffalo adapter (found .buffalo.dev.yml)")
		return adapters.CreateBuffaloAdapter(host, dir)
	}

	fmt.Println("[app]", host, "using the static adapter")
	return adapters.CreateStaticAdapter(dir)
}

func (a *App) idleMonitor() error {
	fmt.Println("[app]", a.Host, "starting idle monitor")
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if a.idle() {
				a.Stop("app is idle", nil)
				return nil
			}
		}
	}

}

func (a *App) idle() bool {
	diff := time.Since(a.LastUsed)
	if diff > 60*60*time.Second {
		return true
	}

	return false
}

func findAppForHost(host string) (*App, error) {
	hostParts := strings.Split(host, ":")
	host = hostParts[0]

	lock.Lock()
	if apps == nil {
		apps = make(map[string]*App)
	}

	app := apps[host]
	lock.Unlock()

	if app != nil {
		return app, nil
	}

	fmt.Println("[app]", host, "creating app")

	app, err := NewApp(host)
	if err != nil {
		fmt.Println("[app]", host, "error creating app", err)
		return nil, errors.Context(err, "app failed to create")
	}

	err = app.Start()
	if err != nil {
		fmt.Println("[app]", host, "error starting app", err)
		app.Stop("app failed to start", err)
		return nil, errors.Context(err, "app failed to start")
	}

	fmt.Println("[app]", host, "created app")
	// TODO: apps should be keyed by Dir not host as you might have multiple
	// hosts pointing to the same app
	lock.Lock()
	apps[host] = app
	lock.Unlock()

	return app, nil
}
