package control

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/moomerman/zap/zap"
)

// fakeManager has one app, phx.test, and records the actions it is sent
type fakeManager struct {
	mu      sync.Mutex
	status  zap.State
	actions []string
	events  chan zap.Event
}

func (m *fakeManager) snapshot() zap.Snapshot {
	m.mu.Lock()
	defer m.mu.Unlock()
	return zap.Snapshot{
		Key:     "/code/phx",
		Status:  m.status,
		Config:  zap.AppConfig{Host: "phx.test", Dir: "/code/phx", Command: "mix phx.server"},
		Started: time.Now().Add(-time.Minute),
	}
}

func (m *fakeManager) List() []zap.Snapshot { return []zap.Snapshot{m.snapshot()} }

func (m *fakeManager) Get(host string) (zap.Snapshot, error) {
	if host != "phx.test" {
		return zap.Snapshot{}, fmt.Errorf("%w: %s", zap.ErrNotFound, host)
	}
	return m.snapshot(), nil
}

func (m *fakeManager) act(name string, status zap.State) func(string) error {
	return func(key string) error {
		m.mu.Lock()
		defer m.mu.Unlock()
		m.actions = append(m.actions, name+" "+key)
		if status != "" {
			m.status = status
		}
		return nil
	}
}

func (m *fakeManager) Start(key string) error   { return m.act("start", zap.StateRunning)(key) }
func (m *fakeManager) Stop(key string) error    { return m.act("stop", zap.StateStopped)(key) }
func (m *fakeManager) Restart(key string) error { return m.act("restart", zap.StateRunning)(key) }
func (m *fakeManager) StartNgrok(key string) error {
	return errors.New("ngrok is not installed")
}

func (m *fakeManager) WriteLog(key string, w io.Writer) error {
	io.WriteString(w, "booting\nready\n")
	return nil
}

func (m *fakeManager) Subscribe() (<-chan zap.Event, func()) { return m.events, func() {} }

func startServer(t *testing.T, m Manager) (*Client, *Server) {
	t.Helper()
	// unix socket paths are limited to ~104 bytes, which t.TempDir can exceed
	dir, err := os.MkdirTemp("", "zap")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	path := filepath.Join(dir, "zapd.sock")

	s := NewServer(m)
	done := make(chan error, 1)
	go func() { done <- s.ListenAndServe(path) }()
	t.Cleanup(func() {
		s.Shutdown(context.Background())
		if err := <-done; err != http.ErrServerClosed {
			t.Errorf("expected ErrServerClosed, got %v", err)
		}
	})

	for i := 0; ; i++ {
		if _, err := os.Stat(path); err == nil {
			break
		}
		if i > 100 {
			t.Fatal("socket was not created")
		}
		time.Sleep(10 * time.Millisecond)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("expected socket to be 0600, got %v", info.Mode().Perm())
	}
	return NewClient(path), s
}

func TestAppsAndActions(t *testing.T) {
	m := &fakeManager{status: zap.StateStopped, events: make(chan zap.Event)}
	c, _ := startServer(t, m)
	ctx := context.Background()

	apps, err := c.Apps(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(apps) != 1 || apps[0].Host != "phx.test" || apps[0].Kind != "server" || apps[0].Status != "stopped" {
		t.Fatalf("unexpected apps %+v", apps)
	}
	if apps[0].Started == nil || apps[0].LastUsed != nil {
		t.Errorf("expected only started to be set, got %v %v", apps[0].Started, apps[0].LastUsed)
	}

	app, err := c.Start(ctx, "phx.test")
	if err != nil {
		t.Fatal(err)
	}
	if app.Status != "running" {
		t.Errorf("expected running after start, got %q", app.Status)
	}
	if _, err := c.Restart(ctx, "phx.test"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Stop(ctx, "phx.test"); err != nil {
		t.Fatal(err)
	}
	want := []string{"start /code/phx", "restart /code/phx", "stop /code/phx"}
	if strings.Join(m.actions, ",") != strings.Join(want, ",") {
		t.Errorf("expected actions %v, got %v", want, m.actions)
	}

	var log strings.Builder
	if err := c.Log(ctx, "phx.test", &log); err != nil {
		t.Fatal(err)
	}
	if log.String() != "booting\nready\n" {
		t.Errorf("unexpected log %q", log.String())
	}
}

func TestErrors(t *testing.T) {
	c, _ := startServer(t, &fakeManager{events: make(chan zap.Event)})
	ctx := context.Background()

	if _, err := c.App(ctx, "nope.test"); err == nil || !strings.Contains(err.Error(), "app not found") {
		t.Errorf("expected not found, got %v", err)
	}
	if _, err := c.Ngrok(ctx, "phx.test"); err == nil || err.Error() != "ngrok is not installed" {
		t.Errorf("expected the manager's error, got %v", err)
	}
}

func TestEvents(t *testing.T) {
	m := &fakeManager{events: make(chan zap.Event, 3)}
	c, _ := startServer(t, m)

	m.events <- zap.Event{Type: zap.EventLog, Key: "/code/other", Host: "other.test", Line: "skipped\n"}
	m.events <- zap.Event{Type: zap.EventStatus, Key: "/code/phx", Host: "phx.test", Status: zap.StateRunning}
	m.events <- zap.Event{Type: zap.EventLog, Key: "/code/phx", Host: "phx.test", Line: "GET /\n"}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var got []Event
	stop := errors.New("stop")
	err := c.Events(ctx, "phx.test", func(e Event) error {
		got = append(got, e)
		if len(got) == 2 {
			return stop
		}
		return nil
	})
	if err != stop {
		t.Fatalf("expected the stream to end with our error, got %v", err)
	}
	if got[0].Type != "status" || got[0].Status != "running" || got[1].Type != "log" || got[1].Line != "GET /\n" {
		t.Errorf("unexpected events %+v", got)
	}
}

func TestEventStreamsEndOnShutdown(t *testing.T) {
	c, s := startServer(t, &fakeManager{events: make(chan zap.Event)})

	done := make(chan error, 1)
	go func() {
		done <- c.Events(context.Background(), "", func(Event) error { return nil })
	}()
	time.Sleep(100 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown waited for the event stream: %v", err)
	}
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("event stream did not end")
	}
}

func TestStaleSocketIsReplaced(t *testing.T) {
	dir, err := os.MkdirTemp("", "zap")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, "zapd.sock")

	if err := os.WriteFile(path, nil, 0600); err != nil {
		t.Fatal(err)
	}
	l, err := listen(path)
	if err != nil {
		t.Fatalf("expected a stale socket to be replaced, got %v", err)
	}
	defer l.Close()

	if _, err := listen(path); err == nil {
		t.Error("expected a socket in use to be refused")
	}
}
