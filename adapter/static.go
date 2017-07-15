package adapter

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"time"
)

type StaticAdapter struct {
	Host string
	Dir  string
}

func CreateStaticAdapter(host, dir string) (Adapter, error) {
	return &StaticAdapter{
		Host: host,
		Dir:  dir,
	}, nil
}

func (d *StaticAdapter) Start() error         { return nil }
func (d *StaticAdapter) Stop() error          { return nil }
func (d *StaticAdapter) Command() *exec.Cmd   { return nil }
func (d *StaticAdapter) WriteLog(w io.Writer) {}

func (d *StaticAdapter) Serve(w http.ResponseWriter, r *http.Request) {
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
