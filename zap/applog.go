package zap

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// maxLogSize is the size above which an app's log is rotated when it starts
const maxLogSize = 10 * 1024 * 1024

// appLog receives an app's output. It writes to <dir>/<host>.log, or to
// zapd's own stdout when there is no log dir.
type appLog struct {
	mu     sync.Mutex
	w      io.Writer
	closer io.Closer
	prefix string
}

func openAppLog(dir, host string) *appLog {
	if dir == "" {
		return &appLog{w: os.Stdout, prefix: "  [log] " + host + ": "}
	}

	f, err := openLogFile(dir, host)
	if err != nil {
		log.Println("[app]", host, "unable to open log file, logging to stdout", err)
		return &appLog{w: os.Stdout, prefix: "  [log] " + host + ": "}
	}

	fmt.Fprintf(f, "\n=== zap starting %s at %s ===\n", host, time.Now().Format(time.RFC3339))
	return &appLog{w: f, closer: f}
}

func openLogFile(dir, host string) (*os.File, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	path := filepath.Join(dir, filepath.Base(host)+".log")
	if info, err := os.Stat(path); err == nil && info.Size() > maxLogSize {
		if err := os.Rename(path, path+".1"); err != nil {
			log.Println("[app]", host, "unable to rotate log", err)
		}
	}

	return os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
}

// WriteLine writes a line of output
func (l *appLog) WriteLine(line string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.w != nil {
		io.WriteString(l.w, l.prefix+line)
	}
}

// Close closes the log file. Later writes are dropped.
func (l *appLog) Close() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closer != nil {
		l.closer.Close()
	}
	l.w = nil
}
