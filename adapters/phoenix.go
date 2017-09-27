package adapters

import "regexp"

const phoenixShellCommand = `exec bash -c '
exec mix do deps.get, phx.server'
`

// CreatePhoenixAdapter creates a new phoenix adapter
func CreatePhoenixAdapter(host, dir string) (Adapter, error) {

	// TODO: look at the mix.exs file and determine which version of phoenix
	// we're starting and use the correct start command
	mixFileChanged, nil := regexp.Compile("You must restart your server")

	return &AppProxyAdapter{
		Name:            "Phoenix",
		Host:            host,
		Dir:             dir,
		RestartPatterns: []*regexp.Regexp{mixFileChanged},
		EnvPortName:     "PHX_PORT",
		shellCommand:    phoenixShellCommand,
	}, nil
}
