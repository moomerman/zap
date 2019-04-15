package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
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

func init() {
	os.Setenv("GODEBUG", os.Getenv("GODEBUG")+",tls13=1")
}

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

	server := &zap.Server{
		HTTPAddr:  *fHTTP,
		HTTPSAddr: *fHTTPS,
	}

	go func() {
		ch := make(chan os.Signal)
		signal.Notify(ch, os.Interrupt, os.Kill, syscall.SIGTERM)

		log.Printf("[zap] caught signal '%v' shutting down\n", <-ch)
		responder.Stop()
		server.Stop()
	}()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		responder.Serve()
	}()

	go func() {
		defer wg.Done()
		server.Serve()
	}()

	wg.Wait()
}
