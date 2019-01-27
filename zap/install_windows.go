package zap

func installService(httpAddr, httpsAddr string) error {
	return nil
}

func uninstallService() error {
	return nil
}

// symbolic links via powershell (requires admin)
// New-Item -path ~\.zap\mysite.test -itemType SymbolicLink -target ~\Workspace\mysite
