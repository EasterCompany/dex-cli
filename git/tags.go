package git

import (
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"
)

// GetLatestTag retrieves the highest semantic version tag from a git repository.
// It lists all tags, filters for semantic versions, sorts them, and returns the highest.
// If no tags are found, it returns a default of "v0.0.0" so the build process can proceed.
func GetLatestTag(repoPath string) (string, error) {
	// List all tags
	cmd := exec.Command("git", "tag", "-l")
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git tag -l failed in %s: %w\nOutput: %s", repoPath, err, string(output))
	}

	// Parse tags into semantic versions
	tagsOutput := strings.TrimSpace(string(output))
	if tagsOutput == "" {
		return "v0.0.0", nil
	}

	tags := strings.Split(tagsOutput, "\n")
	var semverTags []string

	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		// Check if it's a semantic version (X.Y.Z or vX.Y.Z)
		cleanTag := strings.TrimPrefix(tag, "v")
		parts := strings.Split(cleanTag, ".")
		if len(parts) == 3 {
			// Verify all parts are numbers
			if _, err := strconv.Atoi(parts[0]); err == nil {
				if _, err := strconv.Atoi(parts[1]); err == nil {
					if _, err := strconv.Atoi(parts[2]); err == nil {
						semverTags = append(semverTags, tag)
					}
				}
			}
		}
	}

	if len(semverTags) == 0 {
		return "v0.0.0", nil
	}

	// Sort tags by semantic version (highest first)
	sort.Slice(semverTags, func(i, j int) bool {
		vi := parseVersion(semverTags[i])
		vj := parseVersion(semverTags[j])
		if vi[0] != vj[0] {
			return vi[0] > vj[0] // Major
		}
		if vi[1] != vj[1] {
			return vi[1] > vj[1] // Minor
		}
		return vi[2] > vj[2] // Patch
	})

	return semverTags[0], nil
}

// parseVersion parses a version string like "1.2.3" or "v1.2.3" into [major, minor, patch]
func parseVersion(version string) [3]int {
	version = strings.TrimPrefix(version, "v")
	parts := strings.Split(version, ".")
	var result [3]int
	for i := 0; i < 3 && i < len(parts); i++ {
		result[i], _ = strconv.Atoi(parts[i])
	}
	return result
}
