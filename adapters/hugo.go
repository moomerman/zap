package adapters

const hugoShellCommand = `exec bash -c '
exec hugo server -D -p %s -b https://%s/ --appendPort=false --liveReloadPort=443 --navigateToChanged'
`

// CreateHugoAdapter creates a new hugo adapter
func CreateHugoAdapter(host, dir string) (Adapter, error) {
	return &AppProxyAdapter{
		Host:         host,
		Dir:          dir,
		ShellCommand: hugoShellCommand,
	}, nil
}
