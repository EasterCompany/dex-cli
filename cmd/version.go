package cmd

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func Version(version, branch, commit, buildDate, buildYear string) {
	// Style for the trademark part
	darkStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240")) // Dark grey

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

	// Style the trademark part
	trademarkPart := darkStyle.Render(fmt.Sprintf("| Easter Company™ © %s", buildYear))

	// Assemble and print the final output
	fmt.Printf("%s %s\n", fullVersion, trademarkPart)
}
