package zap

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/moomerman/zap/adapter"
	"github.com/moomerman/zap/adapter/proxy"
	"github.com/moomerman/zap/ngrok"
	"github.com/vektra/errors"
)

var appsMu sync.Mutex
var apps map[string]*app

// app holds the state of a running Application
type app struct {
	Config *AppConfig

	lastUsedMu sync.Mutex
	LastUsed   time.Time

	adapterMu sync.Mutex
	Adapter   adapter.Adapter

	Started time.Time
	Ngrok   *ngrok.Tunnel
}

// newApp creates a new App with the given configuration
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
	a.adapterMu.Lock()
	a.adapterMu.Unlock()

	var adpt adapter.Adapter
	var err error

	if a.Config.Dir != "" {
		adpt, err = GetAdapter(a.Config.Host, a.Config.Dir)
		if err != nil {
			return errors.Context(err, "could not determine adapter")
		}
	} else {
		adpt, err = proxy.New(a.Config.Host, a.Config.Content)
		if err != nil {
			return errors.Context(err, "unable to create proxy adapter")
		}
	}

	a.Adapter = adpt
	return nil
}

// Start starts an application and monitors activity
func (a *app) Start() error {
	a.adapterMu.Lock()
	defer a.adapterMu.Unlock()

	err := a.Adapter.Start()
	if err != nil {
		return err
	}

	a.LastUsed = time.Now()

	go a.idleMonitor()
	return nil
}

// Stop stops an application handler and removes the app
func (a *app) Stop(reason string, e error) error {
	a.adapterMu.Lock()
	defer a.adapterMu.Unlock()
	appsMu.Lock()
	defer appsMu.Unlock()

	log.Println("[app]", a.Config.Host, "stopping", reason, e)
	delete(apps, a.Config.Key)
	return a.Adapter.Stop(errors.Context(e, reason))
}

// Restart restarts an application adapter
func (a *app) RestartAdapter() error {
	a.adapterMu.Lock()
	defer a.adapterMu.Unlock()

	if err := a.Adapter.Stop(errors.New("requested restart")); err != nil {
		log.Println("[app]", a.Config.Host, "error stopping adapter on restart", err)
	}
	if err := a.newAdapter(); err != nil {
		return err
	}
	return a.Start()
}

// Status returns the status of the application
func (a *app) Status() string {
	return string(a.Adapter.Status())
}

func (a *app) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.lastUsedMu.Lock()
	a.LastUsed = time.Now()
	a.lastUsedMu.Unlock()
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

func (a *app) StartNgrok(host string, port int) error {
	// TODO: check if another ngrok instance exists
	// if so, stop it and cleanup
	ngrok, err := ngrok.StartTunnel(host, port)
	if err != nil {
		return err
	}

	a.Ngrok = ngrok

	// TODO: add the symbolic link

	return nil
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
	a.lastUsedMu.Lock()
	defer a.lastUsedMu.Unlock()

	diff := time.Since(a.LastUsed)
	if diff > 60*60*time.Second {
		return true
	}

	return false
}

func findAppForHost(host string) (*app, error) {
	appsMu.Lock()
	defer appsMu.Unlock()

	host = strings.Split(host, ":")[0]

	config, err := getAppConfig(host)
	if err != nil {
		return nil, err
	}

	if apps == nil {
		apps = make(map[string]*app)
	}

	app := apps[config.Key]

	if app != nil {
		if app.Status() == "stopped" {
			if err := app.RestartAdapter(); err != nil {
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

	apps[config.Key] = app

	return app, nil
}
