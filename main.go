package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/moomerman/zap/dns"
	"github.com/moomerman/zap/zap"
)

var (
	fInstall    = flag.Bool("install", false, "Install the server")
	fUninstall  = flag.Bool("uninstall", false, "Uninstall the server")
	fHTTP       = flag.String("http", "127.0.0.1:80", "address to listen on for HTTP requests")
	fHTTPS      = flag.String("https", "127.0.0.1:443", "address to listen on for HTTPS requests")
	fDNS        = flag.String("dns", "127.0.0.1:9253", "address to listen on for DNS requests")
	fDNSDomains = flag.String("domains", "dev:test", "domains to handle for DNS requests, separate with :")
)

func main() {
	flag.Parse()

	if *fInstall {
		if err := zap.Install(*fHTTP, *fHTTPS, *fDNS); err != nil {
			log.Fatal("[zap] unable to install zap", err)
		}
		return
	}

	if *fUninstall {
		if err := zap.Uninstall(); err != nil {
			log.Fatal("[zap] unable to uninstall zap", err)
		}
		return
	}

	responder := &dns.Responder{
		Address: *fDNS,
		Domains: strings.Split(*fDNSDomains, ":"),
	}
	go responder.Serve()

	server := &zap.Server{
		HTTPAddr:  *fHTTP,
		HTTPSAddr: *fHTTPS,
	}
	server.Serve()

	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)

	log.Println("[zap] shutting down", <-ch)
	responder.Stop()
	server.Stop()
}
