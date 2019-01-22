package zap

func installService(httpAddr, httpsAddr string) error {
	return nil
}

func uninstallService() error {
	return nil
}

// sudo setcap 'cap_net_bind_service=+ep' zapd
// ./zapd -dns 127.0.0.54:53

// /etc/systemd/resolved.conf
// DNS=127.0.0.54
