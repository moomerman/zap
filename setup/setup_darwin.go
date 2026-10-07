package setup

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/moomerman/zap/dns"
	"github.com/moomerman/zap/launchd"
	"github.com/moomerman/zap/zap"
)

func installService(httpAddr, httpsAddr, dnsAddr string, domains []string) error {
	args := []string{
		"-http=" + httpAddr,
		"-https=" + httpsAddr,
		"-dns=" + dnsAddr,
		"-domains=" + strings.Join(domains, ":"),
	}
	return launchd.Install(appID, args, filepath.Join(zap.DefaultLogDir(), appName+".log"))
}

func uninstallService() error {
	return launchd.Uninstall(appID)
}

// installResolver writes /etc/resolver/<domain> for each domain, which needs
// root, so it goes through sudo
func installResolver(dnsAddr string, domains []string) error {
	body, err := dns.ResolverConfig(dnsAddr)
	if err != nil {
		return fmt.Errorf("resolver for %s: %w", dnsAddr, err)
	}

	for _, domain := range domains {
		path := filepath.Join(dns.ResolverDir, domain)
		existing, err := os.ReadFile(path)
		if err == nil && string(existing) == body {
			continue
		}
		if err == nil && !dns.IsZapResolver(existing) {
			return fmt.Errorf("%s exists and wasn't written by zap, move it aside and run -install again", path)
		}

		fmt.Printf("* Writing %s (sudo may ask for your password)\n", path)
		if err := asRoot(nil, "mkdir", "-p", dns.ResolverDir); err != nil {
			return err
		}
		if err := asRoot(strings.NewReader(body), "tee", path); err != nil {
			return err
		}
	}

	conflicts, err := dns.ResolverConflicts(dns.ResolverDir, domains)
	if err != nil {
		return err
	}
	for domain, paths := range conflicts {
		fmt.Printf("! %s also claim .%s and macOS may send .%s lookups there instead of to zap, remove them with: sudo rm %s\n", strings.Join(paths, ", "), domain, domain, strings.Join(paths, " "))
	}

	return nil
}

// uninstallResolver removes the resolver files zap wrote for each domain
func uninstallResolver(domains []string) error {
	for _, domain := range domains {
		path := filepath.Join(dns.ResolverDir, domain)
		existing, err := os.ReadFile(path)
		if err != nil || !dns.IsZapResolver(existing) {
			continue
		}

		fmt.Printf("* Removing %s (sudo may ask for your password)\n", path)
		if err := asRoot(nil, "rm", "-f", path); err != nil {
			return err
		}
	}
	return nil
}

// asRoot runs a command as root, through sudo unless zapd already is root
func asRoot(stdin io.Reader, name string, args ...string) error {
	if os.Geteuid() != 0 {
		args = append([]string{name}, args...)
		name = "sudo"
	}
	cmd := exec.Command(name, args...)
	cmd.Stdin = stdin
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %w", strings.Join(cmd.Args, " "), err)
	}
	return nil
}
