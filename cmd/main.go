package main

import (
	"flag"
	"log"

	"github.com/moomerman/phx-dev/dev"
	"github.com/moomerman/phx-dev/devcert"
)

var (
	fInstall   = flag.Bool("install", false, "Install the server")
	fUninstall = flag.Bool("uninstall", false, "Uninstall the server")
	fLaunchd   = flag.Bool("launchd", false, "Server is running via launchd")
)

var httpPort = "8080"
var httpsPort = "443"

func main() {
	flag.Parse()

	if *fInstall {
		err := devcert.CreateCert()
		if err != nil {
			log.Fatal("Unable to install self-signed certificate", err)
		}

		err = dev.Install(httpPort, httpsPort)
		if err != nil {
			log.Fatal("Unable to install daemon", err)
		}

		return
	}

	if *fUninstall {
		dev.Uninstall()
		return
	}

	if *fLaunchd {
		httpPort = "Socket"
		httpsPort = "SocketTLS"
	} else {
		httpPort = ":" + httpPort
		httpsPort = ":" + httpsPort
	}

	server := dev.NewServer()

	server.ServeTLS(httpsPort)
}
