package zap

import "github.com/moomerman/zap/cert"

const appID = "com.github.moomerman.zap"
const appName = "zapd"

// Install installs zap
func Install(httpAddr, httpsAddr, dnsAddr string) error {

	// TODO: install the dns resolver

	if err := installCertificate(); err != nil {
		return err
	}

	return installService(httpAddr, httpsAddr)
}

// Uninstall removes zap
func Uninstall() error {
	return uninstallService()
}

func installCertificate() error {
	return cert.CreateCert()
}
