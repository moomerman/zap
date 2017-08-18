package adapters

const buffaloShellCommand = `exec bash -c '
exec buffalo dev'
`

// CreateBuffaloAdapter creates a new buffalo adapter
func CreateBuffaloAdapter(host, dir string) (Adapter, error) {
	return &AppProxyAdapter{
		Host:         host,
		Dir:          dir,
		ShellCommand: buffaloShellCommand,
		EnvPortName:  "PORT",
		readyChan:    make(chan struct{}),
	}, nil
}
