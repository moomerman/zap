package adapters

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"time"
)

// StaticAdapter holds the state for the application
type StaticAdapter struct {
	Dir    string
	status Status
}

// CreateStaticAdapter creates a new static HTML application
func CreateStaticAdapter(dir string) (Adapter, error) {
	return &StaticAdapter{
		Dir: dir,
	}, nil
}

// Status returns the status of the adapter
func (d *StaticAdapter) Status() Status {
	return d.status
}

// ServeHTTP implements the http.Handler interface
func (d *StaticAdapter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	filename := d.Dir + r.URL.Path

	info, err := os.Stat(filename)

	if err != nil {
		fmt.Println("[static]", fullURL(r), "->", 404)
		http.Error(w, "404 Not Found", http.StatusNotFound)
		return
	}

	if info.IsDir() {
		filename = path.Join(filename, "index.html")
		info, err = os.Stat(filename)
		if err != nil || info.IsDir() {
			fmt.Println("[static]", fullURL(r), "->", 404)
			http.Error(w, "404 Not Found", http.StatusNotFound)
			return
		}
	}

	file, err := os.Open(filename)
	if err != nil {
		fmt.Println("[static]", fullURL(r), "->", 500)
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	fmt.Println("[static]", fullURL(r), "->", filename)
	http.ServeContent(w, r, filename, time.Now(), file)
}

// Start doesn't do anything
func (d *StaticAdapter) Start() error {
	d.status = StatusRunning
	return nil
}

// Stop doesn't do anything
func (d *StaticAdapter) Stop() error {
	d.status = StatusStopped
	return nil
}

// Command doesn't do anything
func (d *StaticAdapter) Command() *exec.Cmd { return nil }

// WriteLog doesn't do anything
func (d *StaticAdapter) WriteLog(w io.Writer) {}
