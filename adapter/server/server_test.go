package server

import (
	"errors"
	"testing"
	"time"

	zadapter "github.com/moomerman/zap/adapter"
)

func waitForStatus(t *testing.T, a zadapter.Adapter, want zadapter.Status) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for a.Status() != want {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %q, status is %q", want, a.Status())
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// A start that fails before the process exists must not crash.
func TestStartFailureDoesNotPanic(t *testing.T) {
	t.Setenv("SHELL", "/nonexistent/shell")

	a := New(&Config{Host: "fail.test", Dir: t.TempDir(), ShellCommand: "exec true # %s %s"})

	if err := a.Start(); err == nil {
		t.Fatal("expected start to fail")
	}

	if status := a.Status(); status != zadapter.StatusError {
		t.Errorf("expected status %q, got %q", zadapter.StatusError, status)
	}

	if err := a.Stop(errors.New("test")); err != nil {
		t.Errorf("unexpected error stopping: %v", err)
	}
}
