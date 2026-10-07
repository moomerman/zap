package zap

import (
	"testing"
	"time"
)

// RestartAdapter used to re-lock adapterMu inside newAdapter and Start,
// deadlocking the app (and every request, via findAppForHost).
func TestRestartAdapterDoesNotDeadlock(t *testing.T) {
	a, err := newApp(&AppConfig{Host: "static.test", Dir: t.TempDir(), Key: "static"})
	if err != nil {
		t.Fatal(err)
	}
	if err := a.Start(); err != nil {
		t.Fatal(err)
	}

	done := make(chan error, 1)
	go func() { done <- a.RestartAdapter() }()

	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("RestartAdapter deadlocked")
	}

	if status := a.Status(); status != "running" {
		t.Errorf("expected running after restart, got %q", status)
	}
}
