package buffalo

import (
	"github.com/moomerman/zap/adapter"
	"github.com/moomerman/zap/adapter/server"
)

// New creates a new buffalo adapter
func New(host, dir string) (adapter.Adapter, error) {
	return &server.Adapter{
		Name:         "Buffalo",
		Host:         host,
		Dir:          dir,
		EnvPortName:  "PORT",
		ShellCommand: "exec buffalo dev # %s %s",
	}, nil
}
