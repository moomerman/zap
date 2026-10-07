package server

import (
	"errors"
	"testing"

	zadapter "github.com/moomerman/zap/adapter"
)

// A start that fails before the process exists must not crash in stop().
func TestStartFailureDoesNotPanic(t *testing.T) {
	t.Setenv("SHELL", "/nonexistent/shell")

	a := New(&Config{Host: "fail.test", Dir: t.TempDir(), ShellCommand: "exec true # %s %s"})

	if err := a.Start(); err == nil {
		t.Fatal("expected start to fail")
	}

	if status := a.Status(); status != zadapter.StatusStopped {
		t.Errorf("expected status %q, got %q", zadapter.StatusStopped, status)
	}

	if err := a.Stop(errors.New("test")); err != nil {
		t.Errorf("unexpected error stopping: %v", err)
	}
}
