package server

import "os/exec"

func setProcessGroup(cmd *exec.Cmd) {}

func terminate(cmd *exec.Cmd) error {
	return cmd.Process.Kill()
}

func kill(cmd *exec.Cmd) error {
	return cmd.Process.Kill()
}
