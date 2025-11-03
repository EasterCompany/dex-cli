package cmd

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func Version(version, branch, commit, buildDate, buildYear string) {
	// Define styles
	darkStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))    // Dark grey
	brightStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("86"))   // Bright purple
	dividerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240")) // Dark grey for the dividers

	// Format the build date: 2025-11-03T02:38:13Z -> 2025-11-03-02-38-13
	formattedDate := strings.ReplaceAll(buildDate, "T", "-")
	formattedDate = strings.ReplaceAll(formattedDate, ":", "-")
	formattedDate = strings.TrimSuffix(formattedDate, "Z")

	// Format the architecture: linux/amd64 -> linux_amd64
	arch := fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH)

	// Create the full version string
	fullVersion := fmt.Sprintf("v%s.%s.%s.%s.%s",
		strings.TrimPrefix(version, "v"),
		branch,
		commit,
		formattedDate,
		arch,
	)

	// Assemble the styled parts
	part1 := darkStyle.Render("dex-cli")
	part2 := brightStyle.Render(fullVersion)
	part3 := darkStyle.Render(fmt.Sprintf("Easter Company™ © %s", buildYear))
	divider := dividerStyle.Render("|")

	// Print the final, styled string
	fmt.Printf("%s %s %s %s %s\n", part1, divider, part2, divider, part3)
}
