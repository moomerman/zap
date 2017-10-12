package adapter

import (
	"io"
	"net/http"
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
