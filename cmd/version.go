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
		fmt.Printf("%s\n", fullVersion)
	}
}

// FormatVersion constructs the full version string from build-time variables.
func FormatVersion(version, branch, commit, buildDate, buildHash string) string {
	// Check if version is already fully formatted (has 8 parts separated by dots)
	parts := strings.Split(version, ".")
	if len(parts) >= 8 {
		// Already fully formatted (likely from `dex build`)
		return version
	}

	// Not fully formatted - build it from the individual components
	// Format: major.minor.patch.branch.commit.buildDate.arch.buildHash
	arch := fmt.Sprintf("%s-%s", runtime.GOOS, runtime.GOARCH)
	versionClean := strings.TrimPrefix(version, "v")

	return fmt.Sprintf("%s.%s.%s.%s.%s.%s",
		versionClean, branch, commit, buildDate, arch, buildHash)
}
