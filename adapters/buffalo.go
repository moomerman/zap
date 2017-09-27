package adapters

const buffaloShellCommand = `exec bash -c '
exec buffalo dev'
`

// CreateBuffaloAdapter creates a new buffalo adapter
func CreateBuffaloAdapter(host, dir string) (Adapter, error) {
	return &AppProxyAdapter{
		Name:         "Buffalo",
		Host:         host,
		Dir:          dir,
		EnvPortName:  "PORT",
		shellCommand: buffaloShellCommand,
	}, nil
}
