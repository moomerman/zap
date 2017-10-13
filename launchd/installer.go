package launchd

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/kardianos/osext"
	"github.com/puma/puma-dev/homedir"
	"github.com/vektra/errors"
)

// Install installs the launch agent on macOS
func Install(appID, appName, httpHost, httpPort, tlsHost, tlsPort string) error {
	Uninstall(appID, appName)

	binPath, err := osext.Executable()
	if err != nil {
		return errors.Context(err, "calculating executable path")
	}

	fmt.Printf("* Use '%s' as the location of %s\n", binPath, appName)

	var userTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
   <key>Label</key>
   <string>%s</string>
   <key>ProgramArguments</key>
   <array>
	     <string>%s</string>
			 <string>-launchd</string>
			 <string>-http=Socket</string>
			 <string>-https=SocketTLS</string>
   </array>
   <key>KeepAlive</key>
   <true/>
   <key>RunAtLoad</key>
   <true/>
   <key>Sockets</key>
   <dict>
       <key>Socket</key>
       <dict>
           <key>SockNodeName</key>
           <string>%s</string>
           <key>SockServiceName</key>
           <string>%s</string>
       </dict>
       <key>SocketTLS</key>
       <dict>
           <key>SockNodeName</key>
           <string>%s</string>
           <key>SockServiceName</key>
           <string>%s</string>
       </dict>
   </dict>
   <key>StandardOutPath</key>
   <string>%s</string>
   <key>StandardErrorPath</key>
   <string>%s</string>
</dict>
</plist>
`

	logPath := homedir.MustExpand("~/Library/Logs/" + appName + ".log")
	plistDir := homedir.MustExpand("~/Library/LaunchAgents")
	plist := homedir.MustExpand("~/Library/LaunchAgents/" + appID + ".plist")

	config := []byte(fmt.Sprintf(userTemplate, appName, binPath, httpHost, httpPort, tlsHost, tlsPort, logPath, logPath))

	if err := os.MkdirAll(plistDir, 0644); err != nil {
		return errors.Context(err, "creating LaunchAgents directory")
	}

	if err := ioutil.WriteFile(plist, config, 0644); err != nil {
		return errors.Context(err, "writing LaunchAgent plist")
	}

	exec.Command("launchctl", "unload", plist).Run()

	if err := exec.Command("launchctl", "load", plist).Run(); err != nil {
		return errors.Context(err, "launchctl load <plist>")
	}

	fmt.Printf("* Installed %s on ports: http %s, https %s\n", appID, httpPort, tlsPort)

	return nil
}

// Uninstall removes the launch agent on macOS
func Uninstall(appID, appName string) error {
	plist := homedir.MustExpand("~/Library/LaunchAgents/" + appID + ".plist")

	if err := exec.Command("launchctl", "unload", plist).Run(); err != nil {
		return errors.Context(err, "launchctl unload <plist>")
	}

	if err := os.Remove(plist); err != nil {
		return errors.Context(err, "removing LaunchAgent plist")
	}

	fmt.Printf("* Removed %s from automatically running\n", appID)
	return nil
}
