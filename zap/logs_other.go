//go:build !darwin

package zap

// DefaultLogDir is empty outside macOS, so app output goes to stdout
func DefaultLogDir() string {
	return ""
}
