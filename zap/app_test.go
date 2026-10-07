package zap

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// configs maps hosts to app configs for tests
type configs map[string]*AppConfig

func (c configs) resolve(host string) (*AppConfig, error) {
	config, ok := c[host]
	if !ok {
		return nil, os.ErrNotExist
	}
	copy := *config
	copy.Host = host
	return &copy, nil
}

func newTestManager(t *testing.T, c configs) *Manager {
	t.Helper()
	m := newManager(c.resolve)
	t.Cleanup(func() { m.Shutdown(context.Background()) })
	return m
}

func waitFor(t *testing.T, m *Manager, host string, want State) Snapshot {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for {
		s, err := m.Get(host)
		if err != nil {
			t.Fatal(err)
		}
		if s.Status == want {
			return s
		}
		if time.Now().After(deadline) {
			t.Fatalf("%s: timed out waiting for %q, status is %q (%s)", host, want, s.Status, s.Error)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func withTimeout(t *testing.T, name string, f func() error) {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- f() }()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	case <-time.After(10 * time.Second):
		t.Fatalf("%s deadlocked", name)
	}
}

func TestStaticAppLifecycle(t *testing.T) {
	m := newTestManager(t, configs{"static.test": {Dir: t.TempDir(), Key: "static"}})

	if _, err := m.ensure("static.test:443"); err != nil {
		t.Fatal(err)
	}
	waitFor(t, m, "static.test", StateRunning)

	// restart used to deadlock on the app's adapter mutex
	withTimeout(t, "restart", func() error { return m.Restart("static") })
	waitFor(t, m, "static.test", StateRunning)

	withTimeout(t, "stop", func() error { return m.Stop("static") })
	waitFor(t, m, "static.test", StateStopped)

	// a request starts it again
	if _, err := m.ensure("static.test"); err != nil {
		t.Fatal(err)
	}
	waitFor(t, m, "static.test", StateRunning)
}

func TestUnknownApp(t *testing.T) {
	m := newTestManager(t, configs{})

	if _, err := m.ensure("nope.test"); !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
	if err := m.Restart("nope"); !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestServerAppCrashAndRestart(t *testing.T) {
	if testing.Short() {
		t.Skip("starts processes")
	}
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("needs python3")
	}
	t.Setenv("SHELL", "/bin/sh")

	dir := t.TempDir()
	marker := filepath.Join(dir, "crash")
	// listen until the marker file appears, then exit
	command := `python3 -c "
import os, socket, sys, time
s = socket.socket(); s.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
s.bind(('127.0.0.1', int(os.environ['PORT']))); s.listen()
while not os.path.exists('` + marker + `'): time.sleep(0.05)
sys.exit(3)"`

	m := newTestManager(t, configs{"server.test": {Dir: dir, Command: command, Port: "PORT", Scheme: "http", Key: "server"}})
	events, unsubscribe := m.Subscribe()
	defer unsubscribe()

	if _, err := m.ensure("server.test"); err != nil {
		t.Fatal(err)
	}
	s := waitFor(t, m, "server.test", StateRunning)
	if s.Adapter.Pid == 0 || s.Adapter.Port == "" {
		t.Errorf("expected pid and port in snapshot, got %+v", s.Adapter)
	}

	if err := os.WriteFile(marker, nil, 0644); err != nil {
		t.Fatal(err)
	}
	s = waitFor(t, m, "server.test", StateError)
	if s.Error == "" {
		t.Error("expected an error describing the crash")
	}

	// a crashed app restarts on the next request (this used to deadlock)
	os.Remove(marker)
	withTimeout(t, "ensure", func() error { _, err := m.ensure("server.test"); return err })
	waitFor(t, m, "server.test", StateRunning)

	withTimeout(t, "shutdown", func() error { return m.Shutdown(context.Background()) })
	waitFor(t, m, "server.test", StateStopped)

	var statuses []State
	for {
		select {
		case e := <-events:
			if e.Type == EventStatus {
				statuses = append(statuses, e.Status)
			}
			continue
		default:
		}
		break
	}
	want := []State{StateStarting, StateRunning, StateError, StateStarting, StateRunning, StateStopping, StateStopped}
	if len(statuses) != len(want) {
		t.Fatalf("expected events %v, got %v", want, statuses)
	}
	for i := range want {
		if statuses[i] != want[i] {
			t.Fatalf("expected events %v, got %v", want, statuses)
		}
	}
}

func TestIdleAppsAreStopped(t *testing.T) {
	m := newTestManager(t, configs{"static.test": {Dir: t.TempDir(), Key: "static"}})

	if _, err := m.ensure("static.test"); err != nil {
		t.Fatal(err)
	}
	waitFor(t, m, "static.test", StateRunning)

	m.stopIdle(time.Now().Add(-time.Hour))
	waitFor(t, m, "static.test", StateRunning)

	m.stopIdle(time.Now().Add(time.Second))
	waitFor(t, m, "static.test", StateStopped)
}

func TestTransitions(t *testing.T) {
	cases := []struct {
		from, to State
		ok       bool
	}{
		{StateStopped, StateStarting, true},
		{StateStopped, StateRunning, false},
		{StateStarting, StateRunning, true},
		{StateRunning, StateStarting, false},
		{StateError, StateRunning, false},
		{StateError, StateStarting, true},
		{StateStopping, StateError, false},
		{StateStopping, StateStopped, true},
	}
	for _, c := range cases {
		if got := canTransition(c.from, c.to); got != c.ok {
			t.Errorf("%s -> %s: expected %v, got %v", c.from, c.to, c.ok, got)
		}
	}
}
