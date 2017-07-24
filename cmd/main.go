package main

import (
	"flag"
	"log"
	"strconv"

	"github.com/moomerman/phx-dev/cert"
	"github.com/moomerman/phx-dev/dev"
)

var (
	fInstall   = flag.Bool("install", false, "Install the server")
	fUninstall = flag.Bool("uninstall", false, "Uninstall the server")
	fLaunchd   = flag.Bool("launchd", false, "Server is running via launchd")
	fHTTPPort  = flag.Int("http-port", 80, "port to listen on for HTTP requests")
	fHTTPSPort = flag.Int("https-port", 443, "port to listen on for HTTPS requests")
)

func main() {
	flag.Parse()

	if *fInstall {
		err := cert.CreateCert()
		if err != nil {
			log.Fatal("Unable to install self-signed certificate", err)
		}

		err = dev.Install(*fHTTPPort, *fHTTPSPort)
		if err != nil {
			log.Fatal("Unable to install daemon", err)
		}

		return
	}

	if *fUninstall {
		dev.Uninstall()
		return
	}

	var httpPort, httpsPort string

	if *fLaunchd {
		httpPort = "Socket"
		httpsPort = "SocketTLS"
	} else {
		httpPort = ":" + strconv.Itoa(*fHTTPPort)
		httpsPort = ":" + strconv.Itoa(*fHTTPSPort)
	}

	server := dev.NewServer()

	go server.Serve(httpPort)
	server.ServeTLS(httpsPort)
}
