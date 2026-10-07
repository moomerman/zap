package zap

import (
	"os"
	"path/filepath"
)

// DefaultLogDir is where zapd and its apps log: ~/Library/Logs/zap
func DefaultLogDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, "Library", "Logs", "zap")
}
