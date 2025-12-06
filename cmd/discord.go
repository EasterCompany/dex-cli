package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/ui"
	"github.com/EasterCompany/dex-cli/utils"
)

func handleDefaultDiscordOutput() error {
	ui.PrintHeader("Event Command Usage")
	ui.PrintInfo("  discord service          | Show the raw status from the /service endpoint")
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
	case "channels":
		return handleDiscordChannels()
	default:
		return fmt.Errorf("unknown discord subcommand: %s\n\nUsage:\n  discord service\n  discord channels", subcommand)
	}
}

type ChannelInfo struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Type     int      `json:"type"`
	Position int      `json:"position"`
	Users    []string `json:"users,omitempty"`
}

type CategoryInfo struct {
	ID       string        `json:"id"`
	Name     string        `json:"name"`
	Position int           `json:"position"`
	Channels []ChannelInfo `json:"channels"`
}

type GuildStructureResponse struct {
	GuildID       string         `json:"guild_id"`
	GuildName     string         `json:"guild_name"`
	Categories    []CategoryInfo `json:"categories"`
	Uncategorized []ChannelInfo  `json:"uncategorized"`
}

func handleDiscordChannels() error {
	def, err := config.Resolve("discord")
	if err != nil {
		return err
	}

	status, _, err := utils.GetHTTPBody(def.GetHTTP("/context/guild"))
	if err != nil {
		return fmt.Errorf("failed to get guild structure: %w", err)
	}

	var guilds []GuildStructureResponse
	if err := json.Unmarshal(status, &guilds); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	for _, guild := range guilds {
		ui.PrintHeader(fmt.Sprintf("Guild: %s (%s)", guild.GuildName, guild.GuildID))

		// Print Uncategorized channels first
		if len(guild.Uncategorized) > 0 {
			for _, ch := range guild.Uncategorized {
				printChannel(ch)
			}
			fmt.Println()
		}

		// Print Categories
		for _, cat := range guild.Categories {
			ui.PrintInfo(fmt.Sprintf("📂 %s", cat.Name))
			for _, ch := range cat.Channels {
				printChannel(ch)
			}
			fmt.Println()
		}
	}

	return nil
}

func printChannel(ch ChannelInfo) {
	icon := "#"
	if ch.Type == 2 { // Voice
		icon = "🔊"
	}

	userText := ""
	if len(ch.Users) > 0 {
		userText = fmt.Sprintf(" (Users: %v)", ch.Users)
	}

	fmt.Printf("  %s %s%s\n", icon, ch.Name, userText)
}
