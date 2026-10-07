package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/moomerman/zap/control"
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
	fLogs       = flag.String("logs", zap.DefaultLogDir(), "directory for app logs, one file per app (empty logs to stdout)")
	fControl    = flag.String("control", control.DefaultSocketPath(), "unix socket for the control API used by the zap command (empty to disable)")
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

	manager := zap.NewManager()
	manager.LogDir = *fLogs

	server := &zap.Server{
		HTTPAddr:  *fHTTP,
		HTTPSAddr: *fHTTPS,
		Manager:   manager,
	}

	controlServer := control.NewServer(manager)

	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, os.Interrupt, syscall.SIGTERM)

		log.Printf("[zap] caught signal '%v' shutting down\n", <-ch)
		responder.Stop()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		controlServer.Shutdown(ctx)
		server.Stop()
	}()

	var wg sync.WaitGroup
	wg.Add(2)

	if *fControl != "" {
		go func() {
			if err := controlServer.ListenAndServe(*fControl); err != http.ErrServerClosed {
				log.Println("[zap] control API stopped", err)
			}
		}()
	}

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
