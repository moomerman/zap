package launchd

import (
	"strings"
	"testing"
)

func TestPlist(t *testing.T) {
	got := string(Plist("com.example.zap", []string{"/Users/me/R&D/zapd", "-http=127.0.0.1:80"}, "/tmp/zapd.log"))

	for _, want := range []string{
		"<key>Label</key>\n\t<string>com.example.zap</string>",
		"<string>/Users/me/R&amp;D/zapd</string>\n\t\t<string>-http=127.0.0.1:80</string>\n\t</array>",
		"<key>StandardOutPath</key>\n\t<string>/tmp/zapd.log</string>",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("plist is missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "Sockets") {
		t.Errorf("plist still asks launchd for sockets:\n%s", got)
	}
}
