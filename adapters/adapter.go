package adapters

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path"
)

// Adapter defines the interface for an Adapter implementation
type Adapter interface {
	Start() error
	Stop(reason error) error
	Status() Status
	WriteLog(io.Writer)
	ServeHTTP(w http.ResponseWriter, r *http.Request)
}

// Status defines the possible states of the adapter
type Status string

const (
	// StatusStarting is the initial state of the adapter
	StatusStarting Status = "starting"
	// StatusRunning is the successful running state of the adapter
	StatusRunning Status = "running"
	// StatusStopping is the state when an adapter is stopping
	StatusStopping Status = "stopping"
	// StatusStopped is the state when an adapter has been stopped
	StatusStopped Status = "stopped"
	// StatusError is the state when an error has occurred
	StatusError Status = "error"
)

// GetAdapter returns the corresponding adapter for the given
// host/dir combination
func GetAdapter(host, dir string) (Adapter, error) {
	_, err := os.Stat(path.Join(dir, "mix.exs"))
	if err == nil {
		log.Println("[app]", host, "using the phoenix adapter (found mix.exs)")
		return CreatePhoenixAdapter(host, dir)
	}

	_, err = os.Stat(path.Join(dir, "Gemfile"))
	if err == nil {
		log.Println("[app]", host, "using the rails adapter (found Gemfile)")
		return CreateRailsAdapter(host, dir)
	}

	_, err = os.Stat(path.Join(dir, ".buffalo.dev.yml"))
	if err == nil {
		log.Println("[app]", host, "using the buffalo adapter (found .buffalo.dev.yml)")
		return CreateBuffaloAdapter(host, dir)
	}

	_, err = os.Stat(path.Join(dir, "config.toml"))
	if err == nil {
		log.Println("[app]", host, "using the hugo adapter (found config.toml)")
		return CreateHugoAdapter(host, dir)
	}

	log.Println("[app]", host, "using the static adapter")
	return CreateStaticAdapter(dir)
}

func findAvailablePort() (string, error) {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		return "", err
	}
	l.Close()

	_, port, err := net.SplitHostPort(l.Addr().String())
	if err != nil {
		return "", err
	}

	return port, nil
}

func fullURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return fmt.Sprint(r.Method, " ", r.Proto, " ", scheme+"://", r.Host, r.URL)
}
