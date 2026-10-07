package zap

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAppOutputGoesToItsOwnLog(t *testing.T) {
	if testing.Short() {
		t.Skip("starts processes")
	}
	t.Setenv("SHELL", "/bin/sh")

	logDir := t.TempDir()
	m := newTestManager(t, configs{"chatty.test": {Dir: t.TempDir(), Command: "echo hello from chatty; sleep 300", Scheme: "http", Key: "chatty"}})
	m.LogDir = logDir

	if _, err := m.ensure("chatty.test"); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(logDir, "chatty.test.log")
	deadline := time.Now().Add(10 * time.Second)
	for {
		data, _ := os.ReadFile(path)
		if strings.Contains(string(data), "hello from chatty\n") {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("expected app output in %s, got %q", path, data)
		}
		time.Sleep(20 * time.Millisecond)
	}

	if err := m.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestAppLogRotatesLargeFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "big.test.log")
	if err := os.WriteFile(path, make([]byte, maxLogSize+1), 0644); err != nil {
		t.Fatal(err)
	}

	l := openAppLog(dir, "big.test")
	l.WriteLine("fresh\n")
	l.Close()
	l.WriteLine("dropped after close\n")

	if info, err := os.Stat(path + ".1"); err != nil || info.Size() != maxLogSize+1 {
		t.Fatalf("expected old log rotated to .1, got %v %v", info, err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(string(data), "fresh\n") {
		t.Errorf("expected new log to end with the written line, got %q", data)
	}
}
