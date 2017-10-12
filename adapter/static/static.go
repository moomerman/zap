package static

import (
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"time"

	"github.com/moomerman/zap/adapter"
)

// Adapter holds the state for the application
type Adapter struct {
	Name    string
	Dir     string
	State   adapter.Status
	BootLog string
}

// New creates a new static HTML application
func New(dir string) (adapter.Adapter, error) {
	return &Adapter{
		Name: "Static",
		Dir:  dir,
	}, nil
}

// Status returns the status of the adapter
func (d *Adapter) Status() adapter.Status {
	return d.State
}

// ServeHTTP implements the http.Handler interface
func (d *Adapter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	filename := d.Dir + r.URL.Path

	info, err := os.Stat(filename)

	if err != nil {
		log.Println("[static]", adapter.FullURL(r), "->", 404)
		http.Error(w, "404 Not Found", http.StatusNotFound)
		return
	}

	if info.IsDir() {
		filename = path.Join(filename, "index.html")
		info, err = os.Stat(filename)
		if err != nil || info.IsDir() {
			log.Println("[static]", adapter.FullURL(r), "->", 404)
			http.Error(w, "404 Not Found", http.StatusNotFound)
			return
		}
	}

	file, err := os.Open(filename)
	if err != nil {
		log.Println("[static]", adapter.FullURL(r), "->", 500)
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	log.Println("[static]", adapter.FullURL(r), "->", filename)
	http.ServeContent(w, r, filename, time.Now(), file)
}

// Start doesn't do anything
func (d *Adapter) Start() error {
	d.State = adapter.StatusRunning
	return nil
}

// Stop doesn't do anything
func (d *Adapter) Stop(reason error) error {
	d.State = adapter.StatusStopped
	return nil
}

// Command doesn't do anything
func (d *Adapter) Command() *exec.Cmd { return nil }

// WriteLog doesn't do anything
func (d *Adapter) WriteLog(w io.Writer) {}
