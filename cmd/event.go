package cmd

import (
	"github.com/EasterCompany/dex-cli/ui"
)

// Event provides commands to interact with the dex-event-service
func Event(args []string) error {
	// Simple test to see if colors render at all.
	ui.PrintInfo("--- ANSI Color Test ---")
	ui.PrintRaw("\x1b[32mThis line should be green.\x1b[0m\n")
	ui.PrintRaw(ui.Colorize("This line should also be green.", ui.ColorGreen) + "\n")
	ui.PrintInfo("--- End Test ---")
	return nil
}
