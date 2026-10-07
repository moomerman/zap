package zap

import (
	"errors"
	"os"

	"github.com/moomerman/zap/cert"
)

const appID = "com.github.moomerman.zap"
const appName = "zapd"

// Install points the system resolver for each domain at the DNS responder,
// creates the certificate authority and installs zapd as a service that
// listens on the given addresses
func Install(httpAddr, httpsAddr, dnsAddr string, domains []string) error {
	if os.Geteuid() == 0 && os.Getenv("SUDO_USER") != "" {
		return errors.New("run -install as yourself, not with sudo; it asks for your password when it needs to")
	}

	if err := installResolver(dnsAddr, domains); err != nil {
		return err
	}

	if err := installCertificate(); err != nil {
		return err
	}

	return installService(httpAddr, httpsAddr, dnsAddr, domains)
}

// Uninstall removes the service and the resolver files for each domain
func Uninstall(domains []string) error {
	// TODO: uninstall the certificate?
	if err := uninstallService(); err != nil {
		return err
	}
	return uninstallResolver(domains)
}

func installCertificate() error {
	return cert.CreateCertLegacy()
}
