package zap

import (
	"github.com/moomerman/zap/launchd"
)

const appID = "com.github.moomerman.zapd"
const appName = "zapd"

// Install installs the launch agent on macOS
func Install(httpPort, tlsPort int) error {
	if err := launchd.Install(appID, appName, httpPort, tlsPort); err != nil {
		return err
	}
	return nil
}

// Uninstall removes the launch agent on macOS
func Uninstall() error {
	if err := launchd.Uninstall(appID, appName); err != nil {
		return err
	}
	return nil
}
