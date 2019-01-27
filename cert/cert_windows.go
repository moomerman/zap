package cert

import "fmt"

const supportDir = "~/.zap/ssl"

// InstallCert installs a CA certificate root in the system cacerts on linux
func InstallCert(cert string) error {
	fmt.Printf("! Add %s to your browser to trust CA\n", cert)
	return nil

	// determine if we're running Ubuntu (and what version) with
	// lsb_release -a
	// cat /etc/*release
}
