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

	// Create the full version string
	fullVersion := fmt.Sprintf("%s.%s.%s.%s.%s.%s",
		version,
		branch,
		commit,
		formattedDate,
		arch,
		buildHash,
	)

	// Create the colored version string
	coloredVersion := fmt.Sprintf("%s%s",
		ui.Colorize("v: ", ui.ColorDarkGray),
		fullVersion,
	)

	// Print version
	fmt.Println(coloredVersion)
}
