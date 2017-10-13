package zap

import (
	"github.com/moomerman/zap/launchd"
)

func installService(httpPort, tlsPort int) error {
	return launchd.Install(appID, appName, httpPort, tlsPort)
}

func uninstallService() error {
	return launchd.Uninstall(appID, appName)
}
