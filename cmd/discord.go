package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/ui"
	"github.com/EasterCompany/dex-cli/utils"
)

func handleDiscordDefault() error {
	ui.PrintHeader("Discord Command Usage")
	ui.PrintInfo("  discord contacts         | Show all guild members and their system levels")
	ui.PrintInfo("  discord channels         | Show the channel structure of connected guilds")
	ui.PrintInfo("  discord service          | Show the raw status from the /service endpoint")
	ui.PrintInfo("  discord quiet [on|off]   | Toggle Dexter's quiet mode for public channels")
	return nil
}

func handleDiscordQuiet(args []string) error {
	if len(args) == 0 {
		options, err := config.LoadOptionsConfig()
		if err != nil {
			return err
		}
		status := "OFF"
		if options.Discord.QuietMode {
			status = "ON"
		}
		ui.PrintInfo(fmt.Sprintf("Discord quiet mode is currently: %s", status))
		return nil
	}

	value := strings.ToLower(args[0])
	var enabled bool
	switch value {
	case "on", "1", "true":
		enabled = true
	case "off", "0", "false":
		enabled = false
	default:
		return fmt.Errorf("invalid value '%s': must be on/off, true/false, or 1/0", args[0])
	}

	options, err := config.LoadOptionsConfig()
	if err != nil {
		return err
	}

	options.Discord.QuietMode = enabled
	if err := config.SaveOptionsConfig(options); err != nil {
		return err
	}

	status := "disabled"
	if enabled {
		status = "enabled"
	}
	ui.PrintSuccess(fmt.Sprintf("Discord quiet mode has been %s", status))
	ui.PrintWarning("Note: Some services may need a restart or reload to apply this change immediately.")
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

func handleDiscordContacts() error {
	def, err := config.Resolve("discord")
	if err != nil {
		return fmt.Errorf("failed to resolve discord service: %w", err)
	}

	ui.PrintHeader("Discord Contacts")
	ui.PrintInfo("Fetching guild members...")

	body, _, err := utils.GetHTTPBody(def.GetHTTP("/contacts"))
	if err != nil {
		return fmt.Errorf("failed to fetch contacts: %w", err)
	}

	var resp struct {
		GuildName string `json:"guild_name"`
		Members   []struct {
			ID       string `json:"id"`
			Username string `json:"username"`
			Level    string `json:"level"`
			Status   string `json:"status"`
		} `json:"members"`
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("failed to parse contacts: %w", err)
	}

	ui.PrintSubHeader(fmt.Sprintf("Server: %s (%d members)", resp.GuildName, len(resp.Members)))

	table := ui.NewTable([]string{"LEVEL", "USERNAME", "STATUS", "DISCORD ID"})

	for _, m := range resp.Members {
		levelColor := ui.ColorDarkGray
		switch m.Level {
		case "Me":
			levelColor = ui.ColorPurple
		case "Master User":
			levelColor = ui.ColorPurple
		case "Admin":
			levelColor = ui.ColorRed
		case "Moderator":
			levelColor = ui.ColorCyan
		case "Contributor":
			levelColor = ui.ColorYellow
		case "User":
			levelColor = ui.ColorWhite
		}

		statusStr := m.Status
		statusColor := ui.ColorDarkGray
		switch m.Status {
		case "online":
			statusColor = ui.ColorGreen
		case "idle":
			statusColor = ui.ColorYellow
		case "dnd":
			statusColor = ui.ColorRed
		}

		levelStr := m.Level
		if m.Level == "Me" {
			levelStr = "DEXTER (ME)"
		}

		table.AddRow([]string{
			ui.Colorize(levelStr, levelColor),
			m.Username,
			ui.Colorize(statusStr, statusColor),
			ui.Colorize(m.ID, ui.ColorDarkGray),
		})
	}

	table.Render()
	return nil
}

func Discord(args []string) error {
	if len(args) == 0 {
		return handleDiscordDefault()
	}

	subcommand := args[0]
	switch subcommand {
	case "contacts":
		return handleDiscordContacts()
	case "channels":
		return handleDiscordChannels()
	case "service":
		return handleDiscordServiceStatus()
	case "quiet":
		return handleDiscordQuiet(args[1:])
	default:
		return handleDiscordDefault()
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
