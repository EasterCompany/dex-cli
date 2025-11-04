package cmd

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/EasterCompany/dex-cli/ui"
)

func Version(version, branch, commit, buildDate, buildYear, buildHash string) {
	// Format the build date: 2025-11-03T02:38:13Z -> 2025-11-03-02-38-13
	formattedDate := strings.ReplaceAll(buildDate, "T", "-")
	formattedDate = strings.ReplaceAll(formattedDate, ":", "-")
	formattedDate = strings.TrimSuffix(formattedDate, "Z")

	// Format the architecture: linux/amd64 -> linux_amd64
	arch := fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH)

	// Split version string into components
	versionParts := strings.Split(strings.TrimPrefix(version, "v"), "-")
	tag := versionParts[0]
	dev := ""
	if len(versionParts) > 1 {
		dev = fmt.Sprintf("-%s", versionParts[1])
	}

	// Create the colored version string
	coloredVersion := fmt.Sprintf("%s %s%s%s%s%s%s%s%s%s%s%s%s",
		ui.Colorize(" v ", ui.ColorDarkGray),
		ui.Colorize(tag, ui.ColorReset), // White
		ui.Colorize(dev, ui.ColorDarkGray),
		ui.Colorize(".", ui.ColorDarkGray),
		ui.Colorize(branch, ui.ColorDarkGray),
		ui.Colorize(".", ui.ColorDarkGray),
		ui.Colorize(commit, ui.ColorReset), // White
		ui.Colorize(".", ui.ColorDarkGray),
		ui.Colorize(formattedDate, ui.ColorDarkGray),
		ui.Colorize(".", ui.ColorDarkGray),
		ui.Colorize(arch, ui.ColorReset), // White
		ui.Colorize(".", ui.ColorDarkGray),
		ui.Colorize(buildHash, ui.ColorDarkGray),
	)

	// Print version
	fmt.Println(coloredVersion)
}
