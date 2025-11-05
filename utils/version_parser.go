package utils

import (
	"github.com/EasterCompany/dex-cli/git"
)

// ParseToShortVersion parses a version string and returns the short version (e.g., "1.2.3").
func ParseToShortVersion(versionStr string) string {
	parsedVersion, err := git.Parse(versionStr)
	if err != nil {
		return "N/A"
	}
	return parsedVersion.Short()
}
