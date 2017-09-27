package adapters

// CreateRailsAdapter creates a new rails adapter
func CreateRailsAdapter(host, dir string) (Adapter, error) {
	return &AppProxyAdapter{
		Name:         "Rails",
		Host:         host,
		Dir:          dir,
		shellCommand: "exec bin/rails s -p %s # %s",
	}, nil
}
