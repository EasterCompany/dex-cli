package utils

import (
	"fmt"
	"os"
	"strings"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/ui"
)

// truncateVersion shortens a version string for display.
func truncateVersion(version string) string {
	// Don't truncate special status strings
	if version == "N/A" || version == "OK" || version == "BAD" {
		return version
	}

	parts := strings.Split(version, ".")
	if len(parts) >= 3 {
		return strings.Join(parts[:3], ".")
	}

	if len(version) > 5 {
		return version[:5]
	}

	return version
}

// FormatSummaryLine formats a single line for the update summary.
func FormatSummaryLine(service config.ServiceDefinition, oldVersion, newVersion string, oldSize, newSize int64) string {
	displayOld := truncateVersion(oldVersion)
	displayNew := truncateVersion(newVersion)

	var versionChange string
	if oldVersion == newVersion {
		versionChange = ui.Colorize(displayNew, ui.ColorDarkGray)
	} else {
		versionChange = ui.Colorize(fmt.Sprintf("%s -> %s", displayOld, displayNew), ui.ColorGreen)
	}

	var sizeChange string
	if oldSize == newSize {
		sizeChange = ui.Colorize(fmt.Sprintf("(%s)", FormatBytes(newSize)), ui.ColorDarkGray)
	} else {
		sizeChange = ui.Colorize(fmt.Sprintf("(%s -> %s)", FormatBytes(oldSize), FormatBytes(newSize)), ui.ColorGreen)
	}

	return fmt.Sprintf("  %-20s %-40s %s", service.ShortName, versionChange, sizeChange)
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
