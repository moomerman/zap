package main

import (
	"flag"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/moomerman/zap/cert"
	"github.com/moomerman/zap/devdns"
	"github.com/moomerman/zap/zap"
)

var (
	fInstall    = flag.Bool("install", false, "Install the server")
	fUninstall  = flag.Bool("uninstall", false, "Uninstall the server")
	fLaunchd    = flag.Bool("launchd", false, "Server is running via launchd")
	fHTTPPort   = flag.Int("http-port", 80, "port to listen on for HTTP requests")
	fHTTPSPort  = flag.Int("https-port", 443, "port to listen on for HTTPS requests")
	fDNSPort    = flag.Int("dns-port", 9253, "port to listen on for DNS requests")
	fDNSDomains = flag.String("domains", "dev:test", "domains to handle for DNS requests, separate with :")
)

func main() {
	flag.Parse()

	if *fInstall {
		err := cert.CreateCert()
		if err != nil {
			log.Fatal("Unable to install self-signed certificate", err)
		}

		err = zap.Install(*fHTTPPort, *fHTTPSPort)
		if err != nil {
			log.Fatal("Unable to install daemon", err)
		}

		return
	}

	if *fUninstall {
		zap.Uninstall()
		return
	}

	domains := strings.Split(*fDNSDomains, ":")
	dns := &devdns.DNSResponder{
		Address: fmt.Sprintf("127.0.0.1:%d", *fDNSPort),
	}
	log.Println("[dns]", "server running at", dns.Address)
	go dns.Serve(domains)

	var httpPort, httpsPort string

	if *fLaunchd {
		httpPort = "Socket"
		httpsPort = "SocketTLS"
	} else {
		httpPort = ":" + strconv.Itoa(*fHTTPPort)
		httpsPort = ":" + strconv.Itoa(*fHTTPSPort)
	}

	server := zap.NewServer()

	go server.Serve(httpPort)
	server.ServeTLS(httpsPort)
}
