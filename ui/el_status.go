package ui

import (
	"fmt"
	"strings"
)

func PrintStatusIndicator(statusType string, message string, details ...string) {
	var emoji string
	var color string

	switch strings.ToLower(statusType) {
	case "success":
		emoji = EmojiSuccess
		color = ColorGreen
	case "failure", "error":
		emoji = EmojiFailure
		color = ColorRed
	case "running", "active":
		emoji = EmojiRunning
		color = ColorCyan
	case "wait", "pending":
		emoji = EmojiWaiting
		color = ColorYellow
	default:
		emoji = EmojiInfo
		color = ColorBlue
	}

	output := fmt.Sprintf("%s %s %s%s%s\n", color, emoji, message, ColorReset, strings.Join(details, " "))

	PrintRaw(output)
}

func PrintSuccessStatus(message string) {
	PrintStatusIndicator("success", message)
}

func PrintFailureStatus(message string) {
	PrintStatusIndicator("failure", message)
}

func PrintRunningStatus(message string) {
	PrintStatusIndicator("running", message)
}

func PrintInfoStatus(message string) {
	PrintStatusIndicator("info", message)
}

// FetchLatestVersion returns the latest version of the CLI.
func FetchLatestVersion() string {
	// In the future, this could fetch the version from a remote server.
	return "v1.0.0" // Placeholder
}
