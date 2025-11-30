package cmd

import (
	"fmt"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/ui"
	"github.com/EasterCompany/dex-cli/utils"
)

func handleDefaultDiscordOutput() error {
	fmt.Println("Discord Command Usage:")
	fmt.Println("  discord service      Show the raw status from the /service endpoint")
	return nil
}

func handleDiscordServiceStatus() error {
	def, err := config.Resolve("discord")
	if err != nil {
		return err
	}

	status, _, err := utils.GetHTTPBody(def.GetHTTP("/service"))
	if err != nil {
		return fmt.Errorf("failed to get discord service status: %w", err)
	}

	ui.PrintCodeBlockFromBytes(status, "discord-service", "json")
	return nil
}

func Discord(args []string) error {
	if len(args) == 0 {
		return handleDefaultDiscordOutput()
	}

	subcommand := args[0]
	switch subcommand {
	case "service":
		return handleDiscordServiceStatus()
	default:
		return fmt.Errorf("unknown discord subcommand: %s\n\nUsage:\n  discord service", subcommand)
	}
}
