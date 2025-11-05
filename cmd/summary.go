package cmd

import (
	"fmt"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/git"
	"github.com/EasterCompany/dex-cli/ui"
)

// formatSummaryLine generates the complete, formatted version comparison line for a service.
func formatSummaryLine(def config.ServiceDefinition, oldVersionStr, newVersionStr string, oldSize, newSize int64) string {
	// --- 1. Format Version Change ---
	oldVersion, errOld := git.Parse(oldVersionStr)
	newVersion, errNew := git.Parse(newVersionStr)
	oldVersionDisplay := formatVersionDisplay(oldVersion, errOld, newVersion, errNew, false)
	newVersionDisplay := formatVersionDisplay(newVersion, errNew, oldVersion, errOld, true)

	// --- 2. Format Git Diff and Size Summary ---
	diffDisplay := ""
	// Only attempt a diff if both versions are valid and have different commits.
	if errOld == nil && errNew == nil && oldVersion.Commit != "" && newVersion.Commit != "" && oldVersion.Commit != newVersion.Commit {
		var summaryParts []string
		sourcePath, err := config.ExpandPath(def.Source)
		if err != nil {
			summaryParts = append(summaryParts, fmt.Sprintf("err: %v", err))
		} else {
			// Git Diff
			stats, err := git.GetDiffSummaryBetween(sourcePath, oldVersion.Commit, newVersion.Commit)
			if err != nil {
				summaryParts = append(summaryParts, fmt.Sprintf("diff err: %v", err))
			} else {
				if stats.FilesChanged > 0 {
					summaryParts = append(summaryParts, fmt.Sprintf("files:%d", stats.FilesChanged))
				}
				if stats.Insertions > 0 {
					summaryParts = append(summaryParts, ui.Colorize(fmt.Sprintf("+%d", stats.Insertions), ui.ColorGreen))
				}
				if stats.Deletions > 0 {
					summaryParts = append(summaryParts, ui.Colorize(fmt.Sprintf("-%d", stats.Deletions), ui.ColorBrightRed))
				}
			}
		}
		// Binary Size Change
		sizeDiff := newSize - oldSize
		if sizeDiff != 0 {
			sign := "+"
			if sizeDiff < 0 {
				sign = "" // The negative sign is already part of the number
			}
			summaryParts = append(summaryParts, fmt.Sprintf("%s%.2fkb", sign, float64(sizeDiff)/1024.0))
		}

		if len(summaryParts) > 0 {
			diffDisplay = ui.Colorize(fmt.Sprintf(" [%s]", ui.Join(summaryParts, "|")), ui.ColorDarkGray)
		}
	}

	// --- 3. Combine and Return ---
	greyV := ui.Colorize("v", ui.ColorDarkGray)
	return fmt.Sprintf("[%s] %s %s -> %s%s", def.ShortName, greyV, oldVersionDisplay, newVersionDisplay, diffDisplay)
}

// formatVersionDisplay creates the colored string for one side of the version arrow (->).
func formatVersionDisplay(v *git.Version, vErr error, other *git.Version, otherErr error, isNewVersion bool) string {
	if vErr != nil {
		return ui.Colorize("N/A", ui.ColorDarkGray)
	}

	shortTag := v.Short()
	// Colorize based on comparison, only if both versions are valid.
	if otherErr == nil {
		comparison := v.Compare(other)
		if isNewVersion && comparison > 0 {
			shortTag = ui.Colorize(shortTag, ui.ColorGreen) // New version is greater
		} else if !isNewVersion && comparison < 0 {
			shortTag = ui.Colorize(shortTag, ui.ColorBrightRed) // Old version is less
		}
	}

	// Append branch and commit, grayed out.
	var branchAndCommit string
	if v.Branch != "" && v.Commit != "" {
		// Shorten commit hash for display
		commit := v.Commit
		if len(commit) > 7 {
			commit = commit[:7]
		}
		branchAndCommit = fmt.Sprintf(".%s.%s", v.Branch, commit)
		branchAndCommit = ui.Colorize(branchAndCommit, ui.ColorDarkGray)
	}

	return fmt.Sprintf("%s%s", shortTag, branchAndCommit)
}
