package main

import (
	"flag"
	"log"

	"github.com/moomerman/phx-dev/dev"
	"github.com/moomerman/phx-dev/devcert"
)

var (
	fInstall = flag.Bool("install", false, "Install the server")
)

func main() {
	flag.Parse()

	if *fInstall {
		err := devcert.CreateCert()
		if err != nil {
			log.Fatal("Unable to install self-signed certificate", err)
		}

		// err := dev.InstallDaemon()
		return
	}

	dev.Start()
}
