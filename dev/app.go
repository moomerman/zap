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

	"github.com/moomerman/phx-dev/adapter"
	"github.com/puma/puma-dev/homedir"
	"github.com/vektra/errors"
)

const appsPath = "~/.phx-dev"

var apps map[string]*App
var lock sync.Mutex

type App struct {
	Host     string
	Link     string
	Dir      string
	LastUsed time.Time

	driver adapter.Adapter
}

func NewApp(host string) (*App, error) {
	path := homedir.MustExpand(appsPath) + "/" + host
	stat, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	var dir string
	var driver adapter.Adapter

	if stat.IsDir() {
		dir, err = os.Readlink(path)
		if err != nil {
			return nil, err
		}

		driver, err = getDriver(host, dir)
		if err != nil {
			return nil, errors.Context(err, "could not determine driver")
		}
	} else {
		fmt.Println("[app]", host, "using the proxy driver")
		// TODO: read the proxy host/port from the file
		// see https://github.com/puma/puma-dev/blob/master/dev/app.go#L473
		driver, err = adapter.CreateProxyAdapter(host, dir, "80")
		if err != nil {
			return nil, errors.Context(err, "unable to create proxy driver")
		}
	}

	return &App{
		Host:   host,
		Link:   path,
		Dir:    dir,
		driver: driver,
	}, nil
}

func (a *App) Start() error {
	err := a.driver.Start()
	if err != nil {
		return err
	}

	go a.idleMonitor()
	return nil
}

func (a *App) Stop(reason string, e error) error {
	fmt.Printf("! Stopping '%s' (%d) %s %s\n", a.Host, a.driver.Command().Process.Pid, reason, e)
	lock.Lock()
	delete(apps, a.Host)
	lock.Unlock()
	return a.driver.Stop()
}

func (a *App) Serve(w http.ResponseWriter, r *http.Request) {
	a.LastUsed = time.Now()
	a.driver.Serve(w, r)
}

func (a *App) WriteLog(w io.Writer) {
	a.driver.WriteLog(w)
}

func getDriver(host, dir string) (adapter.Adapter, error) {
	_, err := os.Stat(path.Join(dir, "mix.exs"))
	if err == nil {
		fmt.Println("[app]", host, "using the phoenix driver (found mix.exs)")
		return adapter.CreatePhoenixAdapter(host, dir)
	}

	fmt.Println("[app]", host, "using the static driver")
	return adapter.CreateStaticAdapter(host, dir)
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
