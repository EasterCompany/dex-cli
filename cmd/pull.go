package cmd

import (
	"fmt"
	"github.com/EasterCompany/dex-cli/ui"
)

// Pull is a dummy function for the 'pull' command
func Pull(args []string) error {
	ui.PrintInfo("Pull command (dummy function) executed.")
	ui.PrintInfo(fmt.Sprintf("Arguments: %v", args))
	return nil
}
