package adapters

// CreateBuffaloAdapter creates a new buffalo adapter
func CreateBuffaloAdapter(host, dir string) (Adapter, error) {
	return &AppProxyAdapter{
		Name:         "Buffalo",
		Host:         host,
		Dir:          dir,
		EnvPortName:  "PORT",
		shellCommand: "exec buffalo dev # %s %s",
	}, nil
}
