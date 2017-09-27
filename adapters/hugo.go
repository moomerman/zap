package adapters

// CreateHugoAdapter creates a new hugo adapter
func CreateHugoAdapter(host, dir string) (Adapter, error) {
	return &AppProxyAdapter{
		Name:         "Hugo",
		Host:         host,
		Dir:          dir,
		shellCommand: "exec hugo server -D -p %s -b https://%s/ --appendPort=false --liveReloadPort=443 --navigateToChanged",
	}, nil
}
