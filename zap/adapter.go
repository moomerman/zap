package zap

import (
	"log"

	"github.com/moomerman/zap/adapter"
	"github.com/moomerman/zap/adapter/server"
	"github.com/moomerman/zap/adapter/static"
)

// GetAdapter returns the corresponding adapter for the given config
func GetAdapter(scheme, host, port, dir, command string) (adapter.Adapter, error) {

	if command != "" {
		config := &server.Config{
			Name:         "Server",
			Scheme:       scheme,
			Host:         host,
			Dir:          dir,
			EnvPortName:  port,
			ShellCommand: "exec " + command + " # %s %s",
		}

		return server.New(config), nil
	}

	log.Println("[app]", host, "using the static adapter")
	return static.New(dir)
}
