package static

import (
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"time"

	zadapter "github.com/moomerman/zap/adapter"
)

// New creates a new static HTML adapter
func New(dir string) (zadapter.Adapter, error) {
	return &adapter{
		Name: "Static",
		Dir:  dir,
	}, nil
}

type adapter struct {
	Name    string
	Dir     string
	State   zadapter.Status
	BootLog string
}

// Status returns the status of the adapter
func (d *adapter) Status() zadapter.Status {
	return d.State
}

// ServeHTTP implements the http.Handler interface
func (d *adapter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	filename := d.Dir + r.URL.Path

	info, err := os.Stat(filename)

	if err != nil {
		log.Println("[static]", zadapter.FullURL(r), "->", 404)
		http.Error(w, "404 Not Found", http.StatusNotFound)
		return
	}

	if info.IsDir() {
		filename = path.Join(filename, "index.html")
		info, err = os.Stat(filename)
		if err != nil || info.IsDir() {
			log.Println("[static]", zadapter.FullURL(r), "->", 404)
			http.Error(w, "404 Not Found", http.StatusNotFound)
			return
		}
	}

	file, err := os.Open(filename)
	if err != nil {
		log.Println("[static]", zadapter.FullURL(r), "->", 500)
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	log.Println("[static]", zadapter.FullURL(r), "->", filename)
	http.ServeContent(w, r, filename, time.Now(), file)
}

// Start doesn't do anything
func (d *adapter) Start() error {
	d.State = zadapter.StatusRunning
	return nil
}

// Stop doesn't do anything
func (d *adapter) Stop(reason error) error {
	d.State = zadapter.StatusStopped
	return nil
}

// Command doesn't do anything
func (d *adapter) Command() *exec.Cmd { return nil }

// WriteLog doesn't do anything
func (d *adapter) WriteLog(w io.Writer) {}
