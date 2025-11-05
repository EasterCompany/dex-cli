package cmd

import (
	"fmt"
	"runtime"
	"strings"
)

// Version prints the formatted version string for the `dex version` command.
func Version(version, branch, commit, buildDate, buildYear, buildHash string) {
	fullVersion := FormatVersion(version, branch, commit, buildDate, buildHash)
	fmt.Printf("v: %s", fullVersion)
}

// FormatVersion constructs the full version string from build-time variables.
func FormatVersion(version, branch, commit, buildDate, buildHash string) string {
	// Format the build date: 2025-11-03T02:38:13Z -> 2025-11-03-02-38-13
	formattedDate := strings.ReplaceAll(buildDate, "T", "-")
	formattedDate = strings.ReplaceAll(formattedDate, ":", "-")
	formattedDate = strings.TrimSuffix(formattedDate, "Z")

	// Format the architecture: linux/amd64 -> linux_amd64
	arch := fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH)

	// Create the full version string, ensuring the 'v' is stripped from the version tag.
	return fmt.Sprintf("%s.%s.%s.%s.%s.%s",
		strings.TrimPrefix(version, "v"),
		branch,
		commit,
		formattedDate,
		arch,
		buildHash,
	)
}
