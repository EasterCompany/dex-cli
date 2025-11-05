package utils

import (
	"fmt"
	"os"
	"strings"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/ui"
)

// SummaryInfo holds all the data needed to print a single service's summary.
type SummaryInfo struct {
	Service       config.ServiceDefinition
	OldVersion    string
	NewVersion    string
	OldSize       int64
	NewSize       int64
	ChangeSummary string
}

// truncateVersion shortens a version string for display.
func truncateVersion(version string) string {
	if version == "N/A" || version == "OK" || version == "BAD" || version == "" {
		return "N/A"
	}
	parts := strings.Split(version, ".")
	if len(parts) >= 3 {
		// Return M.m.p
		return strings.Join(parts[:3], ".")
	}
	return version
}

// PrintSummaryTable prints the final summary table for the build or update commands.
func PrintSummaryTable(summaries []SummaryInfo) {
	table := ui.NewTable([]string{
		"Service",
		"Version Change",
		"Size Change",
		"Code Change Summary",
	})

	for _, s := range summaries {
		// --- Format Version Change ---
		displayOld := truncateVersion(s.OldVersion)
		displayNew := truncateVersion(s.NewVersion)
		var versionChange string
		if s.OldVersion == s.NewVersion {
			versionChange = ui.Colorize(displayNew, ui.ColorDarkGray)
		} else {
			versionChange = ui.Colorize(fmt.Sprintf("%s -> %s", displayOld, displayNew), ui.ColorGreen)
		}

		// --- Format Size Change ---
		sizeDiff := s.NewSize - s.OldSize
		var sizeChange string
		if s.OldSize == 0 && s.NewSize > 0 {
			sizeChange = ui.Colorize(fmt.Sprintf("(+%s)", FormatBytes(s.NewSize)), ui.ColorGreen)
		} else if sizeDiff == 0 {
			sizeChange = ui.Colorize("(0 B)", ui.ColorDarkGray)
		} else {
			sign, color := "+", ui.ColorYellow
			if sizeDiff < 0 {
				sign, color, sizeDiff = "-", ui.ColorGreen, -sizeDiff
			}
			sizeChange = ui.Colorize(fmt.Sprintf("(%s%s)", sign, FormatBytes(sizeDiff)), color)
		}

		// --- Format Change Summary ---
		summary := s.ChangeSummary
		if summary == "" {
			summary = "Not a git repository."
		}
		summary = ui.Colorize(summary, ui.ColorDarkGray)

		table.AddRow(ui.TableRow{
			s.Service.ShortName,
			versionChange,
			sizeChange,
			summary,
		})
	}

	table.Render()
}

// GetBinarySize returns the size of the service's binary in bytes.
func GetBinarySize(service config.ServiceDefinition) int64 {
	path, err := config.ExpandPath(service.GetBinaryPath())
	if err != nil {
		return 0
	}
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}
