package phoenix

import (
	"regexp"

	"github.com/moomerman/zap/adapter"
	"github.com/moomerman/zap/adapter/server"
)

// New creates a new phoenix adapter
func New(scheme, host, dir string) adapter.Adapter {

	mixFileChanged := regexp.MustCompile("You must restart your server")

	config := &server.Config{
		Name:            "Phoenix",
		Scheme:          scheme,
		Host:            host,
		Dir:             dir,
		EnvPortName:     "PHX_PORT",
		ShellCommand:    "exec mix do deps.get, phx.server # %s %s",
		RestartPatterns: []*regexp.Regexp{mixFileChanged},
	}

	return server.New(config)
}
