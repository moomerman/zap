package hugo

import (
	"github.com/moomerman/zap/adapter"
	"github.com/moomerman/zap/adapter/app"
)

// New creates a new hugo adapter
func New(host, dir string) (adapter.Adapter, error) {
	return &app.Adapter{
		Name:         "Hugo",
		Host:         host,
		Dir:          dir,
		ShellCommand: "exec hugo server -D -p %s -b https://%s/ --appendPort=false --liveReloadPort=443 --navigateToChanged",
	}, nil
}
