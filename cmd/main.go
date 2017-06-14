package main

import (
	"flag"
	"log"

	"github.com/moomerman/phx-dev/dev"
	"github.com/moomerman/phx-dev/devcert"
)

var (
	fInstall = flag.Bool("install", false, "Run application installation")
	fStart   = flag.Bool("start", false, "Start the server")
)

func main() {
	flag.Parse()

	if *fInstall {
		err := devcert.CreateCert()
		// dev.InstallDaemon()
		if err != nil {
			log.Fatalf("Unable to install self-signed certificate: %s", err)
		}
		return
	}

	dev.Start()
}
