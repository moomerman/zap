package server

import (
	"io"
	"sync"
)

const defaultLogLines = 1024

// lineBuffer keeps the most recent lines written to it
type lineBuffer struct {
	mu    sync.Mutex
	size  int
	lines []string
	next  int // where the next line goes once the buffer is full
}

func (b *lineBuffer) Append(line string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.size == 0 {
		b.size = defaultLogLines
	}
	if len(b.lines) < b.size {
		b.lines = append(b.lines, line)
		return
	}
	b.lines[b.next] = line
	b.next = (b.next + 1) % b.size
}

// WriteTo writes the buffered lines to w, oldest first
func (b *lineBuffer) WriteTo(w io.Writer) (int64, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	var total int64
	for _, part := range [][]string{b.lines[b.next:], b.lines[:b.next]} {
		for _, l := range part {
			n, err := io.WriteString(w, l)
			total += int64(n)
			if err != nil {
				return total, err
			}
		}
	}
	return total, nil
}
