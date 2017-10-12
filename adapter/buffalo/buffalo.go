package buffalo

import "github.com/moomerman/zap/adapter"
import "github.com/moomerman/zap/adapter/app"

// New creates a new buffalo adapter
func New(host, dir string) (adapter.Adapter, error) {
	return &app.Adapter{
		Name:         "Buffalo",
		Host:         host,
		Dir:          dir,
		EnvPortName:  "PORT",
		ShellCommand: "exec buffalo dev # %s %s",
	}, nil
}
