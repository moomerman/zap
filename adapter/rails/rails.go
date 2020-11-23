package rails

import (
	"github.com/moomerman/zap/adapter"
	"github.com/moomerman/zap/adapter/server"
)

// New creates a new rails adapter
func New(scheme, host, dir string) adapter.Adapter {
	config := &server.Config{
		Name:         "Rails",
		Scheme:       scheme,
		Host:         host,
		Dir:          dir,
		ShellCommand: "exec bin/rails s -p %s # %s",
	}

	return server.New(config)
}
