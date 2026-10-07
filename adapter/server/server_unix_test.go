//go:build !windows

package server

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	zadapter "github.com/moomerman/zap/adapter"
)

// Stop must reach processes the app spawned, not just the shell.
func TestStopKillsProcessGroup(t *testing.T) {
	if testing.Short() {
		t.Skip("starts processes")
	}
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("needs python3")
	}
	t.Setenv("SHELL", "/bin/sh")

	dir := t.TempDir()
	pidFile := filepath.Join(dir, "child.pid")
	command := `sleep 300 & echo $! > ` + pidFile + `; exec python3 -c "
import os, socket, time
s = socket.socket(); s.bind(('127.0.0.1', int(os.environ['PORT']))); s.listen()
time.sleep(300)" # %s %s`

	var statuses []zadapter.Status
	a := New(&Config{
		Host:         "group.test",
		Scheme:       "http",
		Dir:          dir,
		EnvPortName:  "PORT",
		ShellCommand: command,
		OnStatus:     func(s zadapter.Status, err error) { statuses = append(statuses, s) },
	})

	if err := a.Start(); err != nil {
		t.Fatal(err)
	}
	waitForStatus(t, a, zadapter.StatusRunning)

	data, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatal(err)
	}
	var childPid int
	if _, err := fmt.Sscan(strings.TrimSpace(string(data)), &childPid); err != nil {
		t.Fatal(err)
	}

	if err := a.Stop(errors.New("test")); err != nil {
		t.Fatal(err)
	}
	if a.Status() != zadapter.StatusStopped {
		t.Errorf("expected stopped, got %q", a.Status())
	}

	// the orphaned sleep is reparented and reaped by init, so poll for it
	deadline := time.Now().Add(5 * time.Second)
	for syscall.Kill(childPid, 0) == nil {
		if time.Now().After(deadline) {
			syscall.Kill(childPid, syscall.SIGKILL)
			t.Fatal("child process survived stop")
		}
		time.Sleep(20 * time.Millisecond)
	}

	want := []zadapter.Status{zadapter.StatusStarting, zadapter.StatusRunning, zadapter.StatusStopping, zadapter.StatusStopped}
	if strings.Join(toStrings(statuses), ",") != strings.Join(toStrings(want), ",") {
		t.Errorf("expected statuses %v, got %v", want, statuses)
	}
}

func toStrings(statuses []zadapter.Status) []string {
	s := make([]string, len(statuses))
	for i, status := range statuses {
		s[i] = string(status)
	}
	return s
}
