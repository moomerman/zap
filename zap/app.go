package zap

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/moomerman/zap/adapters"
	"github.com/vektra/errors"
)

var apps map[string]*app
var lock sync.Mutex

// app holds the state of a running Application
type app struct {
	Config   *AppConfig
	LastUsed time.Time
	Adapter  adapters.Adapter
	Started  time.Time
}

// newApp creates a new App for the given host
func newApp(config *AppConfig) (*app, error) {
	app := &app{
		Config:  config,
		Started: time.Now(),
	}

	if err := app.newAdapter(); err != nil {
		return nil, err
	}

	return app, nil
}

func (a *app) newAdapter() error {
	var adapter adapters.Adapter
	var err error

	if a.Config.Dir != "" {
		adapter, err = adapters.GetAdapter(a.Config.Host, a.Config.Dir)
		if err != nil {
			return errors.Context(err, "could not determine adapter")
		}
	} else {
		adapter, err = adapters.CreateProxyAdapter(a.Config.Host, a.Config.Content)
		if err != nil {
			return errors.Context(err, "unable to create proxy adapter")
		}
	}

	a.Adapter = adapter
	return nil
}

// Start starts an application and monitors activity
func (a *app) Start() error {
	err := a.Adapter.Start()
	if err != nil {
		return err
	}

	a.LastUsed = time.Now()
	go a.idleMonitor()
	return nil
}

// Stop stops an application
func (a *app) Stop(reason string, e error) error {
	log.Println("[app]", a.Config.Host, "stopping", reason, e)
	lock.Lock()
	delete(apps, a.Config.Key)
	lock.Unlock()
	return a.Adapter.Stop(errors.Context(e, reason))
}

func (a *app) Restart() error {
	if err := a.Adapter.Stop(errors.New("requested restart")); err != nil {
		log.Println("[app]", a.Config.Host, "error stopping adapter on restart", err)
	}
	if err := a.newAdapter(); err != nil {
		return err
	}
	if err := a.Start(); err != nil {
		return err
	}
	return nil
}

// Status returns the status of the application
func (a *app) Status() string {
	return string(a.Adapter.Status())
}

func (a *app) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.LastUsed = time.Now()
	a.Adapter.ServeHTTP(w, r)
}

// WriteLog writes out the application log to the given writer
func (a *app) WriteLog(w io.Writer) {
	a.Adapter.WriteLog(w)
}

// LogTail returns the last X lines of the log file
func (a *app) LogTail() string {
	buf := bytes.NewBufferString("")
	a.WriteLog(buf)
	return buf.String()
}

func (a *app) idleMonitor() {
	log.Println("[app]", a.Config.Host, "starting idle monitor")
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if a.idle() {
				log.Println("[app]", a.Config.Host, "app is idle")
				a.Stop("app is idle", nil)
				return
			}
		}
	}
}

func (a *app) idle() bool {
	diff := time.Since(a.LastUsed)
	if diff > 60*60*time.Second {
		return true
	}

	return false
}

func findAppForHost(host string) (*app, error) {
	hostParts := strings.Split(host, ":")
	host = hostParts[0]

	config, err := getAppConfig(host)
	if err != nil {
		return nil, err
	}

	lock.Lock()
	if apps == nil {
		apps = make(map[string]*app)
	}

	app := apps[config.Key]
	lock.Unlock()

	if app != nil {
		if app.Status() == "stopped" {
			if err := app.Restart(); err != nil {
				return nil, errors.Context(err, "app failed to restart")
			}
		}
		return app, nil
	}

	log.Println("[app]", host, config.Key, "creating app")

	app, err = newApp(config)
	if err != nil {
		log.Println("[app]", host, config.Key, "error creating app", err)
		return nil, errors.Context(err, "app failed to create")
	}

	if err := app.Start(); err != nil {
		log.Println("[app]", host, config.Key, "error starting app", err)
		return nil, errors.Context(err, "app failed to start")
	}

	lock.Lock()
	apps[config.Key] = app
	lock.Unlock()

	return app, nil
}
