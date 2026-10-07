package zap

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/moomerman/zap/adapter"
	"github.com/moomerman/zap/adapter/proxy"
	"github.com/moomerman/zap/adapter/server"
	"github.com/moomerman/zap/adapter/static"
	"github.com/moomerman/zap/ngrok"
)

// State is the lifecycle state of an app
type State = adapter.Status

// App lifecycle states
const (
	StateStarting = adapter.StatusStarting
	StateRunning  = adapter.StatusRunning
	StateStopping = adapter.StatusStopping
	StateStopped  = adapter.StatusStopped
	StateError    = adapter.StatusError
)

// transitions lists the allowed state changes. Anything else is a stale or
// out of order report and is ignored.
var transitions = map[State][]State{
	StateStopped:  {StateStarting},
	StateError:    {StateStarting, StateStopping},
	StateStarting: {StateRunning, StateError, StateStopping},
	StateRunning:  {StateError, StateStopping},
	StateStopping: {StateStopped},
}

func canTransition(from, to State) bool {
	for _, s := range transitions[from] {
		if s == to {
			return true
		}
	}
	return false
}

// app holds the state of an application. The manager owns the app and is the
// only thing that calls start, stop and restart.
//
// Lifecycle operations are serialised by opMu and may block on the adapter.
// mu guards the fields and is never held while calling into the adapter, so
// the adapter can report status changes at any time without deadlocking.
type app struct {
	key     string
	publish func(Event)
	logDir  string

	opMu sync.Mutex

	mu       sync.Mutex
	config   *AppConfig
	state    State
	err      error
	gen      int // incremented on each start, to ignore reports from old adapters
	adapter  adapter.Adapter
	output   *appLog // the current adapter's output, nil if it has none
	started  time.Time
	lastUsed time.Time
	ngrok    *ngrok.Tunnel
	changed  chan struct{} // closed and replaced on every transition
}

func newApp(config *AppConfig, publish func(Event), logDir string) *app {
	return &app{
		key:     config.Key,
		publish: publish,
		logDir:  logDir,
		config:  config,
		state:   StateStopped,
		changed: make(chan struct{}),
	}
}

// Snapshot is a point-in-time copy of an app's state
type Snapshot struct {
	Key      string
	Status   State
	Error    string `json:",omitempty"`
	Config   AppConfig
	Adapter  adapter.Snapshot
	Started  time.Time
	LastUsed time.Time
	Ngrok    *TunnelSnapshot `json:",omitempty"`
}

// TunnelSnapshot describes an ngrok tunnel
type TunnelSnapshot struct {
	URL      string
	AdminURL string
}

func (a *app) snapshot() Snapshot {
	a.mu.Lock()
	s := Snapshot{
		Key:      a.key,
		Status:   a.state,
		Config:   *a.config,
		Started:  a.started,
		LastUsed: a.lastUsed,
	}
	if a.err != nil {
		s.Error = a.err.Error()
	}
	if a.ngrok != nil {
		s.Ngrok = &TunnelSnapshot{URL: a.ngrok.URL, AdminURL: a.ngrok.AdminURL}
	}
	adpt := a.adapter
	a.mu.Unlock()

	if adpt != nil {
		s.Adapter = adpt.Snapshot()
	}
	return s
}

func (a *app) status() State {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.state
}

// waitStarted waits until the app is no longer starting, or ctx is done, and
// returns its state
func (a *app) waitStarted(ctx context.Context) State {
	for {
		a.mu.Lock()
		state, changed := a.state, a.changed
		a.mu.Unlock()

		if state != StateStarting {
			return state
		}
		select {
		case <-changed:
		case <-ctx.Done():
			return state
		}
	}
}

// setConfig updates the config used the next time the app starts
func (a *app) setConfig(config *AppConfig) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.config = config
}

// transition changes state if allowed. The caller must hold mu.
func (a *app) transition(to State, err error) bool {
	if !canTransition(a.state, to) {
		return false
	}
	from := a.state
	a.state = to
	if to == StateError || to == StateStarting {
		a.err = err
	}
	close(a.changed)
	a.changed = make(chan struct{})
	log.Println("[app]", a.config.Host, from, "->", to)
	a.publish(Event{Type: EventStatus, Key: a.key, Host: a.config.Host, Status: to, Error: errString(err), Time: time.Now()})
	return true
}

// start starts the app with a fresh adapter built from the current config
func (a *app) start() error {
	a.opMu.Lock()
	defer a.opMu.Unlock()
	return a.startLocked()
}

func (a *app) startLocked() error {
	a.mu.Lock()
	if !a.transition(StateStarting, nil) {
		a.mu.Unlock()
		return nil // already starting or running
	}
	a.gen++
	gen := a.gen
	config := a.config
	old := a.adapter
	oldOutput := a.output
	a.adapter = nil
	a.output = nil
	a.started = time.Now()
	a.lastUsed = a.started
	a.mu.Unlock()

	// an adapter that failed may still have a process to clean up
	if old != nil {
		old.Stop(fmt.Errorf("replaced"))
	}
	if oldOutput != nil {
		oldOutput.Close()
	}

	adpt, output := a.newAdapter(config, gen)
	a.mu.Lock()
	a.adapter = adpt
	a.output = output
	a.mu.Unlock()

	if err := adpt.Start(); err != nil {
		a.mu.Lock()
		a.transition(StateError, err)
		a.mu.Unlock()
		return err
	}

	// adapters with nothing to boot are running as soon as Start returns
	a.adapterChanged(gen, adpt.Status(), nil)
	return nil
}

// stop stops the app's adapter and any ngrok tunnel
func (a *app) stop(reason error) error {
	a.opMu.Lock()
	defer a.opMu.Unlock()
	return a.stopLocked(reason)
}

func (a *app) stopLocked(reason error) error {
	a.mu.Lock()
	if !a.transition(StateStopping, nil) {
		a.mu.Unlock()
		return nil // already stopped
	}
	adpt := a.adapter
	output := a.output
	a.output = nil
	tunnel := a.ngrok
	a.ngrok = nil
	host := a.config.Host
	a.mu.Unlock()

	log.Println("[app]", host, "stopping:", reason)

	if tunnel != nil {
		tunnel.Stop()
	}

	var err error
	if adpt != nil {
		err = adpt.Stop(reason)
	}
	if output != nil {
		output.Close()
	}

	a.mu.Lock()
	a.transition(StateStopped, nil)
	a.mu.Unlock()
	return err
}

// restart stops the app if needed and starts it again with the current config
func (a *app) restart() error {
	a.opMu.Lock()
	defer a.opMu.Unlock()

	if err := a.stopLocked(fmt.Errorf("requested restart")); err != nil {
		log.Println("[app]", a.key, "error stopping on restart", err)
	}
	return a.startLocked()
}

// adapterChanged applies a status reported by the adapter of generation gen
func (a *app) adapterChanged(gen int, status adapter.Status, err error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if gen != a.gen {
		return
	}

	switch status {
	case adapter.StatusRunning:
		a.transition(StateRunning, nil)
	case adapter.StatusError:
		a.transition(StateError, err)
	case adapter.StatusStopped:
		// a stop we asked for is handled by stopLocked; anything else means
		// the backend went away on its own
		a.transition(StateError, fmt.Errorf("stopped unexpectedly"))
	}
}

// newAdapter builds the adapter for config. Adapters that run a process also
// get an appLog for its output, which the caller must close.
func (a *app) newAdapter(config *AppConfig, gen int) (adapter.Adapter, *appLog) {
	if config.Dir == "" {
		return proxy.New(config.Host, config.Proxy), nil
	}
	if config.Command == "" {
		log.Println("[app]", config.Host, "using the static adapter")
		return static.New(config.Dir), nil
	}

	output := openAppLog(a.logDir, config.Host)
	adpt := server.New(&server.Config{
		Name:         "Server",
		Scheme:       config.Scheme,
		Host:         config.Host,
		Dir:          config.Dir,
		EnvPortName:  config.Port,
		ShellCommand: "exec " + config.Command + " # %s %s",
		OnStatus:     func(status adapter.Status, err error) { a.adapterChanged(gen, status, err) },
		OnLog: func(line string) {
			output.WriteLine(line)
			a.publish(Event{Type: EventLog, Key: a.key, Host: config.Host, Line: line, Time: time.Now()})
		},
	})
	return adpt, output
}

func (a *app) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.touch()

	a.mu.Lock()
	adpt := a.adapter
	a.mu.Unlock()

	if adpt == nil {
		http.Error(w, "502 Bad Gateway", http.StatusBadGateway)
		return
	}
	adpt.ServeHTTP(w, r)
}

// WriteLog writes out the application log to the given writer
func (a *app) WriteLog(w io.Writer) {
	a.mu.Lock()
	adpt := a.adapter
	a.mu.Unlock()

	if adpt != nil {
		adpt.WriteLog(w)
	}
}

// LogTail returns the last X lines of the log file
func (a *app) LogTail() string {
	buf := bytes.NewBufferString("")
	a.WriteLog(buf)
	return buf.String()
}

func (a *app) startNgrok(host string, port int) error {
	a.mu.Lock()
	existing := a.ngrok
	a.mu.Unlock()
	if existing != nil {
		return nil
	}

	tunnel, err := ngrok.StartTunnel(host, port)
	if err != nil {
		return err
	}

	a.mu.Lock()
	a.ngrok = tunnel
	a.mu.Unlock()
	return nil
}

func (a *app) touch() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.lastUsed = time.Now()
}

// idleSince reports whether the app is active and unused since t
func (a *app) idleSince(t time.Time) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	active := a.state == StateStarting || a.state == StateRunning
	return active && a.lastUsed.Before(t)
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
