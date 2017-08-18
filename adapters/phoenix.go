package adapters

import "regexp"

// TODO: handle installing js assets?
// [error] Could not start node watcher because script "/Users/richard/workspace/moocode/hoot/apps/web/assets/node_modules/brunch/bin/brunch" does not exist. Your Phoenix application is still running, however assets won't be compiled. You may fix this by running "cd assets && npm install"

const phoenixShellCommand = `exec bash -c '
cd %s
exec mix do deps.get, phx.server'
`

// CreatePhoenixAdapter creates a new phoenix adapter
func CreatePhoenixAdapter(host, dir string) (Adapter, error) {

	// TODO: look at the mix.exs file and determine which version of phoenix
	// we're starting and use the correct start command
	restart, nil := regexp.Compile("You must restart your server")

	return &AppProxyAdapter{
		Host:            host,
		Dir:             dir,
		ShellCommand:    phoenixShellCommand,
		RestartPatterns: []*regexp.Regexp{restart},
		EnvPortName:     "PHX_PORT",
		readyChan:       make(chan struct{}),
	}, nil
}
