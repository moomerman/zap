package zap

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/kardianos/osext"
	"github.com/puma/puma-dev/homedir"
	"github.com/vektra/errors"
)

const applicationName = "com.github.moomerman.zapd"
const applicationShortName = "zapd"

// Install installs the launch agent on macOS
func Install(httpPort, tlsPort int) error {
	binPath, err := osext.Executable()
	if err != nil {
		return errors.Context(err, "calculating executable path")
	}

	fmt.Printf("* Use '%s' as the location of zap\n", binPath)

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
           <string>0.0.0.0</string>
           <key>SockServiceName</key>
           <string>%d</string>
       </dict>
       <key>SocketTLS</key>
       <dict>
           <key>SockNodeName</key>
           <string>0.0.0.0</string>
           <key>SockServiceName</key>
           <string>%d</string>
       </dict>
   </dict>
   <key>StandardOutPath</key>
   <string>%s</string>
   <key>StandardErrorPath</key>
   <string>%s</string>
</dict>
</plist>
`

	logPath := homedir.MustExpand("~/Library/Logs/" + applicationName + ".log")
	plistDir := homedir.MustExpand("~/Library/LaunchAgents")
	plist := homedir.MustExpand("~/Library/LaunchAgents/" + applicationName + ".plist")

	err = os.MkdirAll(plistDir, 0644)

	if err != nil {
		return errors.Context(err, "creating LaunchAgent directory")
	}

	err = ioutil.WriteFile(
		plist,
		[]byte(fmt.Sprintf(userTemplate, applicationName, binPath, httpPort, tlsPort, logPath, logPath)),
		0644,
	)

	if err != nil {
		return errors.Context(err, "writing LaunchAgent plist")
	}

	exec.Command("launchctl", "unload", plist).Run()

	err = exec.Command("launchctl", "load", plist).Run()
	if err != nil {
		return errors.Context(err, "loading plist into launchctl")
	}

	fmt.Printf("* Installed %s on ports: http %d, https %d\n", applicationShortName, httpPort, tlsPort)

	return nil
}

// Uninstall removes the launch agent on macOS
func Uninstall() {
	plist := homedir.MustExpand("~/Library/LaunchAgents/" + applicationName + ".plist")

	exec.Command("launchctl", "unload", plist).Run()

	os.Remove(plist)

	fmt.Printf("* Removed %s from automatically running\n", applicationShortName)
}
