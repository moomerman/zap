package adapter

import (
	"io"
	"net/http"
)

// Adapter defines the interface for an Adapter implementation
type Adapter interface {
	// Start launches the backend. Adapters that need time to boot return once
	// the backend is launched and report StatusRunning later via their
	// StatusFunc.
	Start() error
	// Stop stops the backend and blocks until it has stopped.
	Stop(reason error) error
	Status() Status
	Snapshot() Snapshot
	WriteLog(io.Writer)
	ServeHTTP(w http.ResponseWriter, r *http.Request)
}

// Snapshot is a point-in-time copy of an adapter's state that is safe to
// read and serialise while the adapter keeps running.
type Snapshot struct {
	Name    string
	Status  Status
	Port    string `json:",omitempty"`
	Pid     int    `json:",omitempty"`
	Command string `json:",omitempty"`
	BootLog string
	Error   string `json:",omitempty"`
}

// StatusFunc is called by an adapter when its status changes asynchronously,
// eg. when a server finishes booting or its process exits.
type StatusFunc func(status Status, err error)

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
