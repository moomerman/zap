//go:build !darwin

package setup

func installService(httpAddr, httpsAddr, dnsAddr string, domains []string) error {
	return nil
}

func uninstallService() error {
	return nil
}

func installResolver(dnsAddr string, domains []string) error {
	return nil
}

func uninstallResolver(domains []string) error {
	return nil
}
