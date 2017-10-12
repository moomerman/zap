package hugo

import (
	"github.com/moomerman/zap/adapter"
	"github.com/moomerman/zap/adapter/server"
)

// New creates a new hugo adapter
func New(host, dir string) adapter.Adapter {
	config := &server.Config{
		Name:         "Hugo",
		Host:         host,
		Dir:          dir,
		ShellCommand: "exec hugo server -D -p %s -b https://%s/ --appendPort=false --liveReloadPort=443 --navigateToChanged",
	}

	return server.New(config)
}
