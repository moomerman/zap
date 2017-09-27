package selfcert

import (
	"fmt"
	"os/exec"
)

const supportDir = "~/Library/Application Support/com.github.moomerman.zap"

func installCert(cert string) error {
	fmt.Printf("* Adding certification to system keychain as trusted\n")
	fmt.Printf("! There is probably a dialog open that you must type your password into\n")

	keychain := "/Library/Keychains/System.keychain"

	command := "do shell script \"security add-trusted-cert -d -r trustRoot -k '" + keychain + "' '" + cert + "'\" with administrator privileges"
	cmd := exec.Command("osascript", "-e", command)
	err := cmd.Run()

	if err != nil {
		return err
	}

	fmt.Printf("* Certificates setup complete\n")

	return nil
}
