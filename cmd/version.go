package cmd

import (
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
)

// VersionInfo holds the structured version information.
type VersionInfo struct {
	Version   string `json:"version"`
	Branch    string `json:"branch"`
	Commit    string `json:"commit"`
	BuildDate string `json:"build_date"`
	Arch      string `json:"arch"`
	BuildHash string `json:"build_hash"`
}

// Version prints the formatted version string for the `dex version` command.
func Version(jsonOutput bool, version, branch, commit, buildDate, buildYear, buildHash string) {
	if jsonOutput {
		// Format the architecture: linux/amd64 -> linux_amd64
		arch := fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH)

		info := VersionInfo{
			Version:   strings.TrimPrefix(version, "v"),
			Branch:    branch,
			Commit:    commit,
			BuildDate: buildDate,
			Arch:      arch,
			BuildHash: buildHash,
		}

		jsonData, err := json.MarshalIndent(info, "", "  ")
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		fmt.Println(string(jsonData))
	} else {
		fullVersion := FormatVersion(version, branch, commit, buildDate, buildHash)
		fmt.Printf("v: %s\n", fullVersion)
	}
}

// FormatVersion constructs the full version string from build-time variables.
func FormatVersion(version, branch, commit, buildDate, buildHash string) string {
	// The 'version' variable from ldflags is the full 8-part string.
	// We just need to ensure it's correctly formatted.
	// If the version string is already dot-separated, we can trust it.
	if len(strings.Split(version, ".")) == 8 {
		return version
	}

	// Fallback for older builds or local dev where `version` might just be a tag.
	version = strings.TrimPrefix(version, "v")
	versionParts := strings.Split(version, ".")
	if len(versionParts) != 3 {
		versionParts = []string{"0", "0", "0"} // Default if tag is invalid
	}

	// Format the build date: 2025-11-03T02:38:13Z -> 2025-11-03-02-38-13
	formattedDate := strings.ReplaceAll(buildDate, "T", "-")
	formattedDate = strings.ReplaceAll(formattedDate, ":", "-")
	formattedDate = strings.TrimSuffix(formattedDate, "Z")

	// Format the architecture: linux/amd64 -> linux_amd64
	arch := fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH)

	return fmt.Sprintf("%s.%s.%s.%s.%s.%s.%s.%s",
		versionParts[0], versionParts[1], versionParts[2],
		branch,
		commit,
		formattedDate,
		arch,
		buildHash,
	)
}
