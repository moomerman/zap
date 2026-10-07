package homedir

import (
	"path/filepath"
	"testing"
)

func TestExpand(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cases := map[string]string{
		"~":                 home,
		"~/.zap":            filepath.Join(home, ".zap"),
		"~/Library/App Dir": filepath.Join(home, "Library/App Dir"),
		"/etc/zap":          "/etc/zap",
		"~other/zap":        "~other/zap",
		"relative":          "relative",
	}
	for in, want := range cases {
		got, err := Expand(in)
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Errorf("Expand(%q) = %q, want %q", in, got, want)
		}
	}
}
