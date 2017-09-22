package adapters

const hugoShellCommand = `exec bash -c '
exec hugo server -D -p %s -b https://%s/ --appendPort=false'
`

// CreateHugoAdapter creates a new hugo adapter
func CreateHugoAdapter(host, dir string) (Adapter, error) {
	return &AppProxyAdapter{
		Host:         host,
		Dir:          dir,
		ShellCommand: hugoShellCommand,
		readyChan:    make(chan struct{}),
	}, nil
}
