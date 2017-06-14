package devcert

import "fmt"

const supportDir = "~/.puma-dev-ssl"

func installCert(cert string) error {
	fmt.Printf("! Add %s to your browser to trust CA\n", cert)
	return nil
}
