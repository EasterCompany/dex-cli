package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// GetLatestTag retrieves the most recent tag from a git repository.
// It uses `git describe` to find the closest tag. If no tags are found,
// it returns a default of "v0.0.0" so the build process can proceed.
func GetLatestTag(repoPath string) (string, error) {
	cmd := exec.Command("git", "describe", "--tags", "--abbrev=0")
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		// If the error is because no tags were found, we can safely default.
		if strings.Contains(string(output), "no names found") || strings.Contains(string(output), "No names found") {
			return "v0.0.0", nil
		}
		return "", fmt.Errorf("git describe failed in %s: %w\nOutput: %s", repoPath, err, string(output))
	}
	return strings.TrimSpace(string(output)), nil
}
