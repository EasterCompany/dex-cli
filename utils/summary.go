package utils

import (
	"fmt"
	"os"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/ui"
)

// FormatSummaryLine formats a single line for the update summary.
func FormatSummaryLine(service config.ServiceDefinition, oldVersion, newVersion string, oldSize, newSize int64) string {
	var versionChange string
	if oldVersion == newVersion {
		versionChange = ui.Colorize(newVersion, ui.ColorDarkGray)
	} else {
		versionChange = ui.Colorize(fmt.Sprintf("%s -> %s", oldVersion, newVersion), ui.ColorGreen)
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
