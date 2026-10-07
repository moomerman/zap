package zap

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"
)

// ErrNotFound is returned when no app matches the given key or host
var ErrNotFound = errors.New("app not found")

// ErrShutdown is returned once the manager has been shut down
var ErrShutdown = errors.New("manager is shut down")

const (
	defaultIdleTimeout = time.Hour
	idleCheckInterval  = 30 * time.Second
	subscriberBuffer   = 256
)

// EventType distinguishes status changes from log output
type EventType string

// Event types
const (
	EventStatus EventType = "status"
	EventLog    EventType = "log"
)

// Event describes a change to an app
type Event struct {
	Type   EventType
	Key    string
	Host   string
	Status State  `json:",omitempty"`
	Error  string `json:",omitempty"`
	Line   string `json:",omitempty"`
	Time   time.Time
}

// Manager owns the set of apps and their lifecycles. It is the single API
// that the HTTP handlers (and any other client) use to inspect and control
// apps.
type Manager struct {
	// IdleTimeout is how long an app can go without requests before it is
	// stopped
	IdleTimeout time.Duration

	resolve func(host string) (*AppConfig, error)

	mu     sync.Mutex
	apps   map[string]*app
	closed bool

	subMu sync.Mutex
	subs  map[chan Event]struct{}

	done      chan struct{}
	closeOnce sync.Once
}

// NewManager returns a manager that reads app configs from ~/.zap and starts
// its idle reaper
func NewManager() *Manager {
	return newManager(getAppConfig)
}

func newManager(resolve func(string) (*AppConfig, error)) *Manager {
	m := &Manager{
		IdleTimeout: defaultIdleTimeout,
		resolve:     resolve,
		apps:        make(map[string]*app),
		subs:        make(map[chan Event]struct{}),
		done:        make(chan struct{}),
	}
	go m.reap()
	return m
}

// List returns a snapshot of every known app, ordered by key
func (m *Manager) List() []Snapshot {
	apps := m.all()
	snapshots := make([]Snapshot, 0, len(apps))
	for _, a := range apps {
		snapshots = append(snapshots, a.snapshot())
	}
	sort.Slice(snapshots, func(i, j int) bool { return snapshots[i].Key < snapshots[j].Key })
	return snapshots
}

// Get returns a snapshot of the app serving host, without starting it
func (m *Manager) Get(host string) (Snapshot, error) {
	a, _, err := m.lookup(host)
	if err != nil {
		return Snapshot{}, err
	}
	return a.snapshot(), nil
}

// Start starts the app with the given key
func (m *Manager) Start(key string) error {
	a, err := m.find(key)
	if err != nil {
		return err
	}
	if m.isClosed() {
		return ErrShutdown
	}
	return a.start()
}

// Stop stops the app with the given key
func (m *Manager) Stop(key string) error {
	a, err := m.find(key)
	if err != nil {
		return err
	}
	return a.stop(errors.New("requested stop"))
}

// Restart stops and starts the app with the given key
func (m *Manager) Restart(key string) error {
	a, err := m.find(key)
	if err != nil {
		return err
	}
	if m.isClosed() {
		return ErrShutdown
	}
	return a.restart()
}

// Subscribe returns a channel of app events and a function to unsubscribe.
// Events are dropped if the subscriber falls too far behind.
func (m *Manager) Subscribe() (<-chan Event, func()) {
	ch := make(chan Event, subscriberBuffer)

	m.subMu.Lock()
	m.subs[ch] = struct{}{}
	m.subMu.Unlock()

	var once sync.Once
	return ch, func() {
		once.Do(func() {
			m.subMu.Lock()
			delete(m.subs, ch)
			m.subMu.Unlock()
			close(ch)
		})
	}
}

// Shutdown stops every app and refuses to start any more. It returns early
// with the context's error if the apps haven't stopped in time.
func (m *Manager) Shutdown(ctx context.Context) error {
	m.mu.Lock()
	m.closed = true
	m.mu.Unlock()
	m.closeOnce.Do(func() { close(m.done) })

	var wg sync.WaitGroup
	for _, a := range m.all() {
		wg.Add(1)
		go func(a *app) {
			defer wg.Done()
			a.stop(errors.New("zap is shutting down"))
		}(a)
	}

	stopped := make(chan struct{})
	go func() {
		wg.Wait()
		close(stopped)
	}()

	select {
	case <-stopped:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// ensure returns the app for host, starting it if it isn't running. A
// crashed or idle app is started again with the latest config.
func (m *Manager) ensure(host string) (*app, error) {
	a, config, err := m.lookup(host)
	if err != nil {
		return nil, err
	}

	switch a.status() {
	case StateStopped, StateError:
		if m.isClosed() {
			return nil, ErrShutdown
		}
		// apps that share a dir share a process, so the host that starts it
		// is the one substituted into its command
		a.setConfig(config)
		if err := a.start(); err != nil {
			return nil, fmt.Errorf("app failed to start: %w", err)
		}
	}

	return a, nil
}

// lookup resolves the config for host and returns the app for it, creating
// a stopped app if this is the first request
func (m *Manager) lookup(host string) (*app, *AppConfig, error) {
	host = strings.Split(host, ":")[0]

	// config is read from disk, so do it outside the lock
	config, err := m.resolve(host)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %s: %v", ErrNotFound, host, err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	a := m.apps[config.Key]
	if a == nil {
		if m.closed {
			return nil, nil, ErrShutdown
		}
		log.Println("[app]", host, config.Key, "creating app")
		a = newApp(config, m.publish)
		m.apps[config.Key] = a
	}

	return a, config, nil
}

func (m *Manager) find(key string) (*app, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	a := m.apps[key]
	if a == nil {
		return nil, fmt.Errorf("%w: %s", ErrNotFound, key)
	}
	return a, nil
}

func (m *Manager) all() []*app {
	m.mu.Lock()
	defer m.mu.Unlock()

	apps := make([]*app, 0, len(m.apps))
	for _, a := range m.apps {
		apps = append(apps, a)
	}
	return apps
}

func (m *Manager) isClosed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.closed
}

func (m *Manager) publish(e Event) {
	m.subMu.Lock()
	defer m.subMu.Unlock()

	for ch := range m.subs {
		select {
		case ch <- e:
		default:
		}
	}
}

// reap stops apps that haven't served a request within IdleTimeout
func (m *Manager) reap() {
	ticker := time.NewTicker(idleCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.done:
			return
		case <-ticker.C:
			m.stopIdle(time.Now().Add(-m.IdleTimeout))
		}
	}
}

func (m *Manager) stopIdle(cutoff time.Time) {
	for _, a := range m.all() {
		if a.idleSince(cutoff) {
			go a.stop(errors.New("app is idle"))
		}
	}
}
