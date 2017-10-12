package phoenix

import (
	"regexp"

	"github.com/moomerman/zap/adapter"
	"github.com/moomerman/zap/adapter/server"
)

// New creates a new phoenix adapter
func New(host, dir string) (adapter.Adapter, error) {

	// TODO: look at the mix.exs file and determine which version of phoenix
	// we're starting and use the correct start command
	mixFileChanged, nil := regexp.Compile("You must restart your server")

	return &server.Adapter{
		Name:            "Phoenix",
		Host:            host,
		Dir:             dir,
		EnvPortName:     "PHX_PORT",
		ShellCommand:    "exec mix do deps.get, phx.server # %s %s",
		RestartPatterns: []*regexp.Regexp{mixFileChanged},
	}, nil
}
