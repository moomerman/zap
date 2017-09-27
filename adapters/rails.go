package adapters

const railsShellCommand = `exec bash -c '
exec bin/rails s -p %s'
`

// CreateRailsAdapter creates a new rails adapter
func CreateRailsAdapter(host, dir string) (Adapter, error) {
	return &AppProxyAdapter{
		Name:         "Rails",
		Host:         host,
		Dir:          dir,
		shellCommand: railsShellCommand,
	}, nil
}
