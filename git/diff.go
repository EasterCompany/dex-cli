package git

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// DiffStats holds the summary of a git diff.
type DiffStats struct {
	FilesChanged int
	Insertions   int
	Deletions    int
}

// GetDiffSummary calculates the diff between the current HEAD and the previous commit.
func GetDiffSummary(repoPath string) (*DiffStats, error) {
	return GetDiffSummaryBetween(repoPath, "HEAD~1", "HEAD")
}

// GetDiffSummaryBetween calculates the diff between two git refs (e.g., commit hashes, tags).
func GetDiffSummaryBetween(repoPath, oldRef, newRef string) (*DiffStats, error) {
	// --shortstat shows only the summary line of a diff
	cmd := exec.Command("git", "diff", "--shortstat", oldRef, newRef)
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		// This can happen if one of the refs is invalid or doesn't exist.
		// We'll treat it as no changes.
		if strings.Contains(string(output), "bad revision") {
			return &DiffStats{}, nil
		}
		return nil, fmt.Errorf("git diff command failed: %w\nOutput: %s", err, string(output))
	}

	return parseDiffStat(string(output))
}

// parseDiffStat parses the output of "git diff --shortstat".
// e.g., " 1 file changed, 5 insertions(+), 2 deletions(-)"
func parseDiffStat(statLine string) (*DiffStats, error) {
	stats := &DiffStats{}
	statLine = strings.TrimSpace(statLine)

	if statLine == "" {
		return stats, nil // No changes
	}

	// Regex to find the numbers for files, insertions, and deletions
	re := regexp.MustCompile(`(\d+)\s*file(?:s)? changed|(\d+)\s*insertion(?:s)?\(\+\)|(\d+)\s*deletion(?:s)?\(-\)`)
	matches := re.FindAllStringSubmatch(statLine, -1)

	for _, match := range matches {
		if numStr := match[1]; numStr != "" { // Files changed
			stats.FilesChanged, _ = strconv.Atoi(numStr)
		}
		if numStr := match[2]; numStr != "" { // Insertions
			stats.Insertions, _ = strconv.Atoi(numStr)
		}
		if numStr := match[3]; numStr != "" { // Deletions
			stats.Deletions, _ = strconv.Atoi(numStr)
		}
	}

	return stats, nil
}

// GetCommitLogBetween returns a formatted string of commit subjects between two git refs.
func GetCommitLogBetween(repoPath, oldRef, newRef string) (string, error) {
	if oldRef == "" || newRef == "" || oldRef == newRef {
		return "No changes.", nil
	}

	// %s is the subject (commit message title)
	cmd := exec.Command("git", "log", "--pretty=format:- %s", fmt.Sprintf("%s..%s", oldRef, newRef))
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		// If the old ref is not found (e.g., a new service), just show the log for the new ref.
		if strings.Contains(string(output), "bad revision") {
			return GetCommitLogBetween(repoPath, "", newRef)
		}
		return "", fmt.Errorf("git log command failed: %w\nOutput: %s", err, string(output))
	}

	log := strings.TrimSpace(string(output))
	if log == "" {
		return "No changes.", nil
	}

	return log, nil
}
