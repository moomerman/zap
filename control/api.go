// Package control is zapd's local control API. zapd serves it on a unix
// socket and the zap CLI (and later the macOS app) talk to it with Client.
//
// Routes, all JSON unless noted:
//
//	GET  /v1/apps                  every app zapd knows about
//	GET  /v1/apps/{host}           one app
//	POST /v1/apps/{host}/start     start, stop or restart an app
//	POST /v1/apps/{host}/stop
//	POST /v1/apps/{host}/restart
//	POST /v1/apps/{host}/ngrok     open an ngrok tunnel to an app
//	GET  /v1/apps/{host}/log       recent output, as plain text
//	GET  /v1/events[?host=]        server-sent events, one Event per message
//
// Errors are returned as {"error": "..."} with a matching status code.
package control

import (
	"time"

	"github.com/moomerman/zap/zap"
)

// App describes an app
type App struct {
	Key    string `json:"key"`
	Host   string `json:"host"`
	Kind   string `json:"kind"` // server, static or proxy
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`

	Dir     string `json:"dir,omitempty"`
	Command string `json:"command,omitempty"`
	Proxy   string `json:"proxy,omitempty"`
	Port    string `json:"port,omitempty"`
	Pid     int    `json:"pid,omitempty"`

	Started  *time.Time `json:"started,omitempty"`
	LastUsed *time.Time `json:"lastUsed,omitempty"`
	Ngrok    string     `json:"ngrok,omitempty"`
}

// Event is a status change or a line of output from an app
type Event struct {
	Type   string    `json:"type"` // status or log
	Key    string    `json:"key"`
	Host   string    `json:"host"`
	Status string    `json:"status,omitempty"`
	Error  string    `json:"error,omitempty"`
	Line   string    `json:"line,omitempty"`
	Time   time.Time `json:"time"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func newApp(s zap.Snapshot) App {
	a := App{
		Key:      s.Key,
		Host:     s.Config.Host,
		Status:   string(s.Status),
		Error:    s.Error,
		Dir:      s.Config.Dir,
		Command:  s.Config.Command,
		Proxy:    s.Config.Proxy,
		Port:     s.Adapter.Port,
		Pid:      s.Adapter.Pid,
		Started:  timePtr(s.Started),
		LastUsed: timePtr(s.LastUsed),
	}
	switch {
	case s.Config.Dir == "":
		a.Kind = "proxy"
	case s.Config.Command == "":
		a.Kind = "static"
	default:
		a.Kind = "server"
	}
	if s.Ngrok != nil {
		a.Ngrok = s.Ngrok.URL
	}
	return a
}

func newEvent(e zap.Event) Event {
	return Event{
		Type:   string(e.Type),
		Key:    e.Key,
		Host:   e.Host,
		Status: string(e.Status),
		Error:  e.Error,
		Line:   e.Line,
		Time:   e.Time,
	}
}

func timePtr(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}
