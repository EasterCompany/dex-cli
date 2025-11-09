package cmd

import (
	"fmt"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/ui"
	"github.com/EasterCompany/dex-cli/utils"
)

func handleDefaultEventOutput() error {
	fmt.Println("Event Command Usage:")
	fmt.Println("  event              Show default event information (e.g., summary or help)")
	fmt.Println("  event service      Show the raw status from the /service endpoint")
	return nil
}

func handleEventServiceStatus() error {
	def, err := config.Resolve("event")
	if err != nil {
		return err
	}

	status, _, err := utils.GetHTTPBody(def.GetHTTP("/service"))
	if err != nil {
		return fmt.Errorf("failed to get event service status: %w", err)
	}

	ui.PrintCodeBlockFromBytes(status, "event-service", "json")
	return nil
}

func Event(args []string) error {
	if len(args) == 0 {
		// Case 1: `event` (no args) -> default output
		return handleDefaultEventOutput()
	}

	subcommand := args[0]
	switch subcommand {
	case "service":
		// Case 2: `event service` -> shows the status from the /service endpoint
		return handleEventServiceStatus()
	default:
		// Handle unknown subcommands
		return fmt.Errorf("unknown event subcommand: %s\n\nUsage:\n  event\n  event service", subcommand)
	}
}
