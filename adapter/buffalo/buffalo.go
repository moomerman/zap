package buffalo

import (
	"github.com/moomerman/zap/adapter"
	"github.com/moomerman/zap/adapter/server"
)

// New creates a new adapter
func New(host, dir string) adapter.Adapter {
	config := &server.Config{
		Name:         "Buffalo",
		Host:         host,
		Dir:          dir,
		EnvPortName:  "PORT",
		ShellCommand: "exec buffalo dev # %s %s",
	}

	return server.New(config)
}
