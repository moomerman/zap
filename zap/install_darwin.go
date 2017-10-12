package zap

import (
	"github.com/moomerman/zap/launchd"
)

const appID = "com.github.moomerman.zap"
const appName = "zapd"

// Install installs the launch agent on macOS
func Install(httpPort, tlsPort int) error {
	return launchd.Install(appID, appName, httpPort, tlsPort)
}

// Uninstall removes the launch agent on macOS
func Uninstall() error {
	return launchd.Uninstall(appID, appName)
}
