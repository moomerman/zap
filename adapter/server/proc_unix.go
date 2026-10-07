//go:build !windows

package server

import (
	"os/exec"
	"syscall"
)

// setProcessGroup starts the command in its own process group so that stop
// signals reach every process it spawns (watchers, port programs etc.)
func setProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// terminate asks the process group to shut down
func terminate(cmd *exec.Cmd) error {
	return syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
}

// kill forcibly stops the process group
func kill(cmd *exec.Cmd) error {
	return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}
