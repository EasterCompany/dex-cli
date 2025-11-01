package ui

import "os/exec"

// CreateCommand creates a new command to be executed
func CreateCommand(name string, arg ...string) *exec.Cmd {
	return exec.Command(name, arg...)
}
