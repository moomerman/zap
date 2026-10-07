package launchd

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// Install writes a launch agent that runs the current executable with args
// and boots it into the user's GUI session, replacing any earlier install.
// The program's output goes to logPath.
func Install(label string, args []string, logPath string) error {
	binPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("calculating executable path: %w", err)
	}
	if binPath, err = filepath.EvalSymlinks(binPath); err != nil {
		return fmt.Errorf("calculating executable path: %w", err)
	}

	fmt.Printf("* Using '%s' as the location of %s\n", binPath, label)

	plist, err := plistPath(label)
	if err != nil {
		return err
	}

	// boot out whatever the old plist describes, even if its label differs
	if _, err := os.Stat(plist); err == nil {
		launchctl("bootout", domain(), plist)
	}

	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		return fmt.Errorf("creating log directory: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(plist), 0755); err != nil {
		return fmt.Errorf("creating LaunchAgents directory: %w", err)
	}
	if err := os.WriteFile(plist, Plist(label, append([]string{binPath}, args...), logPath), 0644); err != nil {
		return fmt.Errorf("writing LaunchAgent plist: %w", err)
	}

	// bootout can return before the old job has gone, which makes an
	// immediate bootstrap fail, so give it a moment
	for attempt := 0; ; attempt++ {
		err = launchctl("bootstrap", domain(), plist)
		if err == nil || attempt == 10 {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if err != nil {
		return err
	}

	fmt.Printf("* Installed %s\n", plist)
	return nil
}

// Uninstall stops the launch agent and removes its plist
func Uninstall(label string) error {
	plist, err := plistPath(label)
	if err != nil {
		return err
	}

	if _, err := os.Stat(plist); os.IsNotExist(err) {
		fmt.Printf("* %s is not installed\n", label)
		return nil
	}

	launchctl("bootout", domain(), plist)

	if err := os.Remove(plist); err != nil {
		return fmt.Errorf("removing LaunchAgent plist: %w", err)
	}

	fmt.Printf("* Removed %s from automatically running\n", label)
	return nil
}

// Plist returns a launch agent that keeps program running
func Plist(label string, program []string, logPath string) []byte {
	var b bytes.Buffer
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>` + escape(label) + `</string>
	<key>ProgramArguments</key>
	<array>
`)
	for _, arg := range program {
		b.WriteString("\t\t<string>" + escape(arg) + "</string>\n")
	}
	b.WriteString(`	</array>
	<key>KeepAlive</key>
	<true/>
	<key>RunAtLoad</key>
	<true/>
	<key>StandardOutPath</key>
	<string>` + escape(logPath) + `</string>
	<key>StandardErrorPath</key>
	<string>` + escape(logPath) + `</string>
</dict>
</plist>
`)
	return b.Bytes()
}

func plistPath(label string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("finding home directory: %w", err)
	}
	return filepath.Join(home, "Library", "LaunchAgents", label+".plist"), nil
}

// domain is the launchd domain for the current user's GUI session
func domain() string {
	return fmt.Sprintf("gui/%d", os.Getuid())
}

func launchctl(args ...string) error {
	out, err := exec.Command("launchctl", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("launchctl %s %s: %w: %s", args[0], args[1], err, bytes.TrimSpace(out))
	}
	return nil
}

func escape(s string) string {
	var b bytes.Buffer
	xml.EscapeText(&b, []byte(s))
	return b.String()
}
