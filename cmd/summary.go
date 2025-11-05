package cmd

import (
	"fmt"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/git"
	"github.com/EasterCompany/dex-cli/ui"
)

// formatSummaryLine generates the complete, formatted version comparison line for a service.
func formatSummaryLine(def config.ServiceDefinition, oldVersionStr, newVersionStr string) string {
	// --- 1. Format Version Change ---
	oldVersion, errOld := git.Parse(oldVersionStr)
	newVersion, errNew := git.Parse(newVersionStr)

	var oldVersionDisplay, newVersionDisplay string

	// Format old version
	if errOld != nil {
		oldVersionDisplay = ui.Colorize("N/A", ui.ColorDarkGray)
	} else {
		shortTag := oldVersion.Short()
		// Colorize red if it was upgraded
		if errNew == nil && oldVersion.Compare(newVersion) < 0 {
			shortTag = ui.Colorize(shortTag, ui.ColorBrightRed)
		}
		oldVersionDisplay = formatVersionDisplay(oldVersion, shortTag)
	}

	// Format new version
	if errNew != nil {
		newVersionDisplay = ui.Colorize("N/A", ui.ColorDarkGray)
	} else {
		shortTag := newVersion.Short()
		// Colorize green if it was upgraded
		if errOld == nil && newVersion.Compare(oldVersion) > 0 {
			shortTag = ui.Colorize(shortTag, ui.ColorGreen)
		}
		newVersionDisplay = formatVersionDisplay(newVersion, shortTag)
	}

	// --- 2. Format Git Diff Summary ---
	diffDisplay := ""
	// Only show a diff if both versions were parsed correctly and the commits are different.
	if errOld == nil && errNew == nil && oldVersion.Commit != "" && newVersion.Commit != "" && oldVersion.Commit != newVersion.Commit {
		var branch, files, insertions, deletions string

		// Determine branch
		if newVersion.Branch != "" {
			branch = newVersion.Branch
		} else {
			branch = oldVersion.Branch
		}

		sourcePath, err := config.ExpandPath(def.Source)
		if err == nil {
			stats, err := git.GetDiffSummaryBetween(sourcePath, oldVersion.Commit, newVersion.Commit)
			if err == nil {
				if stats.FilesChanged > 0 {
					files = fmt.Sprintf("files:%d", stats.FilesChanged)
				}
				if stats.Insertions > 0 {
					insertions = ui.Colorize(fmt.Sprintf("+%d", stats.Insertions), ui.ColorGreen)
				}
				if stats.Deletions > 0 {
					deletions = ui.Colorize(fmt.Sprintf("-%d", stats.Deletions), ui.ColorBrightRed)
				}
			}
		}

		// If no stats were found, but there was a commit change, show N/A
		if files == "" && insertions == "" && deletions == "" {
			files = "N/A"
		}

		diffParts := []string{branch, files, insertions, deletions}
		diffDisplay = ui.Colorize(fmt.Sprintf(" [%s]", ui.Join(diffParts, "|")), ui.ColorDarkGray)
	}

	// --- 3. Combine and Return ---
	greyV := ui.Colorize("v", ui.ColorDarkGray)
	return fmt.Sprintf("[%s] %s %s -> %s%s", def.ShortName, greyV, oldVersionDisplay, newVersionDisplay, diffDisplay)
}

// formatVersionDisplay combines a parsed version and its pre-formatted tag into a display string.
func formatVersionDisplay(v *git.Version, shortTag string) string {
	var branchAndCommit string
	if v.Branch != "" && v.Commit != "" {
		branchAndCommit = fmt.Sprintf(".%s.%s", v.Branch, v.Commit)
		branchAndCommit = ui.Colorize(branchAndCommit, ui.ColorDarkGray)
	}
	return fmt.Sprintf("%s%s", shortTag, branchAndCommit)
}
