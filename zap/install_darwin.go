package zap

import (
	"net"

	"github.com/moomerman/zap/launchd"
)

func installService(httpAddr, httpsAddr string) error {
	httpHost, httpPort, err := net.SplitHostPort(httpAddr)
	if err != nil {
		return err
	}

	httpsHost, httpsPort, err := net.SplitHostPort(httpsAddr)
	if err != nil {
		return err
	}

	return launchd.Install(appID, appName, httpHost, httpPort, httpsHost, httpsPort)
}

func uninstallService() error {
	return launchd.Uninstall(appID, appName)
}
