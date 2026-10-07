package server

import "bytes"

// lineWriter is an io.Writer that calls line for each complete line written
type lineWriter struct {
	line func(string)
	buf  []byte
}

func (w *lineWriter) Write(p []byte) (int, error) {
	w.buf = append(w.buf, p...)
	for {
		i := bytes.IndexByte(w.buf, '\n')
		if i < 0 {
			break
		}
		w.line(string(w.buf[:i+1]))
		w.buf = w.buf[i+1:]
	}
	return len(p), nil
}

// flush emits any trailing output that didn't end in a newline. It must not
// be called concurrently with Write.
func (w *lineWriter) flush() {
	if len(w.buf) > 0 {
		w.line(string(w.buf) + "\n")
		w.buf = nil
	}
}
