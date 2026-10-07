package zap

import (
	"context"
	"errors"
	"io"
	"net/http"
	"sync"
	"testing"

	"github.com/moomerman/zap/adapter"
)

// fakeAdapter reports status the way the server adapter does, but only
// becomes running or fails when the test says so
type fakeAdapter struct {
	onStatus adapter.StatusFunc
	startErr error

	mu      sync.Mutex
	status  adapter.Status
	started int
	stops   []string
}

func (f *fakeAdapter) Start() error {
	f.mu.Lock()
	f.started++
	f.mu.Unlock()
	if f.startErr != nil {
		f.set(adapter.StatusError, f.startErr)
		return f.startErr
	}
	f.set(adapter.StatusStarting, nil)
	return nil
}

func (f *fakeAdapter) Stop(reason error) error {
	f.mu.Lock()
	f.stops = append(f.stops, reason.Error())
	f.mu.Unlock()
	f.set(adapter.StatusStopping, nil)
	f.set(adapter.StatusStopped, nil)
	return nil
}

// set changes the status and reports it, as an adapter does from its own
// goroutines
func (f *fakeAdapter) set(status adapter.Status, err error) {
	f.mu.Lock()
	f.status = status
	f.mu.Unlock()
	f.onStatus(status, err)
}

func (f *fakeAdapter) Status() adapter.Status {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.status
}

func (f *fakeAdapter) Snapshot() adapter.Snapshot {
	return adapter.Snapshot{Name: "Fake", Status: f.Status()}
}

func (f *fakeAdapter) WriteLog(io.Writer)                           {}
func (f *fakeAdapter) ServeHTTP(http.ResponseWriter, *http.Request) {}

func (f *fakeAdapter) stopReasons() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.stops...)
}

// fakes builds a fakeAdapter for each start and remembers them in order
type fakes struct {
	mu       sync.Mutex
	built    []*fakeAdapter
	startErr error // given to the next adapter built
}

func (fs *fakes) build(config *AppConfig, onStatus adapter.StatusFunc) (adapter.Adapter, *appLog) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	f := &fakeAdapter{onStatus: onStatus, startErr: fs.startErr, status: adapter.StatusStopped}
	fs.startErr = nil
	fs.built = append(fs.built, f)
	return f, nil
}

func (fs *fakes) failNext(err error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	fs.startErr = err
}

func (fs *fakes) count() int {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	return len(fs.built)
}

func (fs *fakes) get(t *testing.T, i int) *fakeAdapter {
	t.Helper()
	fs.mu.Lock()
	defer fs.mu.Unlock()
	if i >= len(fs.built) {
		t.Fatalf("expected at least %d adapters, got %d", i+1, len(fs.built))
	}
	return fs.built[i]
}

const fakeHost = "fake.test"

func newFakeManager(t *testing.T) (*Manager, *fakes) {
	t.Helper()
	fs := &fakes{}
	m := newTestManager(t, configs{fakeHost: {Dir: "/fake", Command: "fake", Key: "fake"}})
	m.build = fs.build
	return m, fs
}

func expectState(t *testing.T, m *Manager, want State) Snapshot {
	t.Helper()
	s, err := m.Get(fakeHost)
	if err != nil {
		t.Fatal(err)
	}
	if s.Status != want {
		t.Fatalf("expected %q, got %q (%s)", want, s.Status, s.Error)
	}
	return s
}

// statusEvents drains the status events received so far
func statusEvents(events <-chan Event) []State {
	var states []State
	for {
		select {
		case e := <-events:
			if e.Type == EventStatus {
				states = append(states, e.Status)
			}
		default:
			return states
		}
	}
}

func expectEvents(t *testing.T, events <-chan Event, want ...State) {
	t.Helper()
	got := statusEvents(events)
	if len(got) != len(want) {
		t.Fatalf("expected events %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected events %v, got %v", want, got)
		}
	}
}

func TestManagerBootsInTheBackground(t *testing.T) {
	m, fs := newFakeManager(t)
	events, unsubscribe := m.Subscribe()
	defer unsubscribe()

	if _, err := m.ensure(fakeHost); err != nil {
		t.Fatal(err)
	}
	expectState(t, m, StateStarting)

	// requests while it boots don't start it again
	if _, err := m.ensure(fakeHost); err != nil {
		t.Fatal(err)
	}
	if err := m.Start("fake"); err != nil {
		t.Fatal(err)
	}
	if fs.count() != 1 || fs.get(t, 0).started != 1 {
		t.Fatalf("expected one adapter started once, got %d adapters", fs.count())
	}

	fs.get(t, 0).set(adapter.StatusRunning, nil)
	expectState(t, m, StateRunning)
	expectEvents(t, events, StateStarting, StateRunning)
}

func TestManagerConcurrentRequestsStartOneAdapter(t *testing.T) {
	m, fs := newFakeManager(t)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := m.ensure(fakeHost); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()

	if n := fs.count(); n != 1 {
		t.Fatalf("expected one adapter, got %d", n)
	}
	expectState(t, m, StateStarting)
}

func TestManagerStartFailure(t *testing.T) {
	m, fs := newFakeManager(t)
	fs.failNext(errors.New("no such command"))

	if _, err := m.ensure(fakeHost); err == nil {
		t.Fatal("expected the start to fail")
	}
	s := expectState(t, m, StateError)
	if s.Error != "no such command" {
		t.Errorf("expected the start error, got %q", s.Error)
	}

	// the next request tries again with a fresh adapter, cleaning up the
	// failed one first
	if _, err := m.ensure(fakeHost); err != nil {
		t.Fatal(err)
	}
	expectState(t, m, StateStarting)
	if reasons := fs.get(t, 0).stopReasons(); len(reasons) != 1 || reasons[0] != "replaced" {
		t.Errorf("expected the failed adapter to be stopped as replaced, got %v", reasons)
	}
	fs.get(t, 1).set(adapter.StatusRunning, nil)
	expectState(t, m, StateRunning)
}

func TestManagerBackendExitsOnItsOwn(t *testing.T) {
	m, fs := newFakeManager(t)
	if _, err := m.ensure(fakeHost); err != nil {
		t.Fatal(err)
	}
	fs.get(t, 0).set(adapter.StatusRunning, nil)

	fs.get(t, 0).set(adapter.StatusStopped, nil)
	s := expectState(t, m, StateError)
	if s.Error != "stopped unexpectedly" {
		t.Errorf("unexpected error %q", s.Error)
	}

	if _, err := m.ensure(fakeHost); err != nil {
		t.Fatal(err)
	}
	if fs.count() != 2 {
		t.Fatalf("expected a new adapter after the exit, got %d", fs.count())
	}
}

func TestManagerIgnoresReportsFromReplacedAdapters(t *testing.T) {
	m, fs := newFakeManager(t)
	if _, err := m.ensure(fakeHost); err != nil {
		t.Fatal(err)
	}
	old := fs.get(t, 0)
	old.set(adapter.StatusRunning, nil)

	if err := m.Restart("fake"); err != nil {
		t.Fatal(err)
	}
	expectState(t, m, StateStarting)

	// the old process dying late must not fail the new one
	old.set(adapter.StatusError, errors.New("old process crashed"))
	old.set(adapter.StatusStopped, nil)
	expectState(t, m, StateStarting)

	fs.get(t, 1).set(adapter.StatusRunning, nil)
	old.set(adapter.StatusStopped, nil)
	expectState(t, m, StateRunning)
}

func TestManagerStopWhileStarting(t *testing.T) {
	m, fs := newFakeManager(t)
	events, unsubscribe := m.Subscribe()
	defer unsubscribe()

	if _, err := m.ensure(fakeHost); err != nil {
		t.Fatal(err)
	}
	if err := m.Stop("fake"); err != nil {
		t.Fatal(err)
	}
	expectState(t, m, StateStopped)

	// an adapter that finishes booting after the stop doesn't revive the app
	fs.get(t, 0).set(adapter.StatusRunning, nil)
	expectState(t, m, StateStopped)

	// stopping again is a no-op
	if err := m.Stop("fake"); err != nil {
		t.Fatal(err)
	}
	if reasons := fs.get(t, 0).stopReasons(); len(reasons) != 1 {
		t.Errorf("expected one stop, got %v", reasons)
	}
	expectEvents(t, events, StateStarting, StateStopping, StateStopped)
}

func TestManagerRestartFromError(t *testing.T) {
	m, fs := newFakeManager(t)
	if _, err := m.ensure(fakeHost); err != nil {
		t.Fatal(err)
	}
	fs.get(t, 0).set(adapter.StatusError, errors.New("boot timeout"))
	expectState(t, m, StateError)

	events, unsubscribe := m.Subscribe()
	defer unsubscribe()

	if err := m.Restart("fake"); err != nil {
		t.Fatal(err)
	}
	fs.get(t, 1).set(adapter.StatusRunning, nil)
	s := expectState(t, m, StateRunning)
	if s.Error != "" {
		t.Errorf("expected the old error to be cleared, got %q", s.Error)
	}
	expectEvents(t, events, StateStopping, StateStopped, StateStarting, StateRunning)
}

func TestManagerShutdown(t *testing.T) {
	m, fs := newFakeManager(t)
	if _, err := m.ensure(fakeHost); err != nil {
		t.Fatal(err)
	}
	fs.get(t, 0).set(adapter.StatusRunning, nil)

	if err := m.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
	expectState(t, m, StateStopped)

	if _, err := m.ensure(fakeHost); !errors.Is(err, ErrShutdown) {
		t.Errorf("expected ErrShutdown from a request, got %v", err)
	}
	if err := m.Start("fake"); !errors.Is(err, ErrShutdown) {
		t.Errorf("expected ErrShutdown from start, got %v", err)
	}
	if err := m.Restart("fake"); !errors.Is(err, ErrShutdown) {
		t.Errorf("expected ErrShutdown from restart, got %v", err)
	}
	if fs.count() != 1 {
		t.Errorf("expected no adapters after shutdown, got %d", fs.count())
	}
}
