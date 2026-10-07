// Package homedir expands a leading ~ in paths to the user's home directory.
package homedir

import (
	"os"
	"path/filepath"
	"strings"
)

// Expand replaces a leading ~ in path with the user's home directory
func Expand(path string) (string, error) {
	if path != "~" && !strings.HasPrefix(path, "~/") {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, path[1:]), nil
}
