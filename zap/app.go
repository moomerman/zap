package zap

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/moomerman/zap/adapters"
	"github.com/vektra/errors"
)

const appsPath = "~/.zap"

var apps map[string]*App
var lock sync.Mutex

// App holds the state of a running Application
type App struct {
	Config   *HostConfig
	LastUsed time.Time
	adapter  adapters.Adapter
	started  time.Time
}

// NewApp creates a new App for the given host
func NewApp(config *HostConfig) (*App, error) {
	var adapter adapters.Adapter
	var err error

	if config.Dir != "" {
		adapter, err = getAdapter(config)
		if err != nil {
			return nil, errors.Context(err, "could not determine adapter")
		}
	} else {
		adapter, err = adapters.CreateProxyAdapter(config.Host, config.Content)
		if err != nil {
			return nil, errors.Context(err, "unable to create proxy adapter")
		}
	}

	return &App{
		Config:  config,
		adapter: adapter,
		started: time.Now(),
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
	fmt.Printf("! Stopping '%s' %s %s\n", a.Config.Host, reason, e)
	lock.Lock()
	delete(apps, a.Config.Key)
	lock.Unlock()
	return a.adapter.Stop()
}

// Status returns the status of the application
func (a *App) Status() string {
	return string(a.adapter.Status())
}

func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.LastUsed = time.Now()
	a.adapter.ServeHTTP(w, r)
}

// WriteLog writes out the application log to the given writer
func (a *App) WriteLog(w io.Writer) {
	a.adapter.WriteLog(w)
}

// LogTail returns the last X lines of the log file
func (a *App) LogTail() string {
	buf := bytes.NewBufferString("")
	a.WriteLog(buf)
	return buf.String()
}

func getAdapter(config *HostConfig) (adapters.Adapter, error) {
	_, err := os.Stat(path.Join(config.Dir, "mix.exs"))
	if err == nil {
		fmt.Println("[app]", config.Host, "using the phoenix adapter (found mix.exs)")
		return adapters.CreatePhoenixAdapter(config.Host, config.Dir)
	}

	_, err = os.Stat(path.Join(config.Dir, "Gemfile"))
	if err == nil {
		fmt.Println("[app]", config.Host, "using the rails adapter (found Gemfile)")
		return adapters.CreateRailsAdapter(config.Host, config.Dir)
	}

	_, err = os.Stat(path.Join(config.Dir, ".buffalo.dev.yml"))
	if err == nil {
		fmt.Println("[app]", config.Host, "using the buffalo adapter (found .buffalo.dev.yml)")
		return adapters.CreateBuffaloAdapter(config.Host, config.Dir)
	}

	_, err = os.Stat(path.Join(config.Dir, "config.toml"))
	if err == nil {
		fmt.Println("[app]", config.Host, "using the hugo adapter (found config.toml)")
		return adapters.CreateHugoAdapter(config.Host, config.Dir)
	}

	fmt.Println("[app]", config.Host, "using the static adapter")
	return adapters.CreateStaticAdapter(config.Dir)
}

func (a *App) idleMonitor() error {
	fmt.Println("[app]", a.Config.Host, "starting idle monitor")
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

	config, err := getHostConfig(host)
	if err != nil {
		return nil, err
	}

	lock.Lock()
	if apps == nil {
		apps = make(map[string]*App)
	}

	app := apps[config.Key]
	lock.Unlock()

	if app != nil {
		if app.Status() == "stopped" {
			err = app.Start()
			if err != nil {
				fmt.Println("[app]", host, config.Key, "error starting app", err)
				app.Stop("app failed to start", err)
				return nil, errors.Context(err, "app failed to start")
			}
		}
		return app, nil
	}

	fmt.Println("[app]", host, config.Key, "creating app")

	app, err = NewApp(config)
	if err != nil {
		fmt.Println("[app]", host, config.Key, "error creating app", err)
		return nil, errors.Context(err, "app failed to create")
	}

	err = app.Start()
	if err != nil {
		fmt.Println("[app]", host, config.Key, "error starting app", err)
		app.Stop("app failed to start", err)
		return nil, errors.Context(err, "app failed to start")
	}

	fmt.Println("[app]", host, config.Key, "created app")

	lock.Lock()
	apps[config.Key] = app
	lock.Unlock()

	return app, nil
}
