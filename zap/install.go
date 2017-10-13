package zap

import "github.com/moomerman/zap/cert"

const appID = "com.github.moomerman.zap"
const appName = "zapd"

// Install installs zap
func Install(httpPort, httpsPort, dnsPort int) error {

	// TODO: install the dns resolver

	if err := installCertificate(); err != nil {
		return err
	}

	return installService(httpPort, httpsPort)
}

// Uninstall removes zap
func Uninstall() error {
	return uninstallService()
}

func installCertificate() error {
	return cert.CreateCert()
}
