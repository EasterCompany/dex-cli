package cmd

import (
	"fmt"
	"github.com/EasterCompany/dex-cli/ui"
	"os"
	"os/exec"
)

// Bun acts as a proxy for the system's bun executable.
func Bun(args []string) error {
	return executeBunCommand("bun", args)
}

// Bunx acts as a proxy for the system's bunx executable.
func Bunx(args []string) error {
	return executeBunCommand("bunx", args)
}

func executeBunCommand(commandName string, args []string) error {
	// Check if bun/bunx exists on system path
	bunPath, err := exec.LookPath(commandName)
	if err != nil {
		ui.PrintError(fmt.Sprintf("%s not found. Please install bun: https://bun.sh/docs/installation", commandName))
		return fmt.Errorf("%s not found", commandName)
	}

	cmd := exec.Command(bunPath, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
