package main

import (
	"flag"
	"log"

	"github.com/moomerman/phx-dev/dev"
)

var (
	fInstall = flag.Bool("install", false, "Run application installation")
	fStart   = flag.Bool("start", false, "Start the server")
)

func main() {
	flag.Parse()

	if *fInstall {
		err := dev.CreateCert()
		// dev.InstallDaemon()
		if err != nil {
			log.Fatalf("Unable to install self-signed certificate: %s", err)
		}
		return
	}

	err := dev.LoadCert()
	if err != nil {
		panic("Unable to load certificate, have you installed yet?")
	}

	dev.Start()
}
