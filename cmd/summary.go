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
	oldVersionDisplay := formatVersionDisplay(oldVersion, errOld)
	newVersionDisplay := formatVersionDisplay(newVersion, errNew)

	// --- 2. Format Git Diff and Size Summary ---
	summaryDisplay := ""
	summaryParts := []string{}
	hasError := false

	// Only attempt a diff if both versions are valid and have different commits.
	if errOld == nil && errNew == nil && oldVersion.Commit != "" && newVersion.Commit != "" && oldVersion.Commit != newVersion.Commit {
		sourcePath, err := config.ExpandPath(def.Source)
		if err != nil {
			summaryParts = append(summaryParts, ui.Colorize(fmt.Sprintf("err: %v", err), ui.ColorBrightRed))
			hasError = true
		} else {
			// Git Diff
			stats, err := git.GetDiffSummaryBetween(sourcePath, oldVersion.Commit, newVersion.Commit)
			if err != nil {
				summaryParts = append(summaryParts, ui.Colorize(fmt.Sprintf("diff err: %v", err), ui.ColorBrightRed))
				hasError = true
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
	}

	// Binary Size Change
	if oldSize > 0 && newSize > 0 {
		sizeDiff := newSize - oldSize
		if sizeDiff != 0 {
			sign := "+"
			color := ui.ColorGreen
			if sizeDiff < 0 {
				sign = "" // The negative sign is already part of the number
				color = ui.ColorBrightRed
			}
			summaryParts = append(summaryParts, ui.Colorize(fmt.Sprintf("%s%.2fkb", sign, float64(sizeDiff)/1024.0), color))
		}
	}

	// Assemble the final summary string
	if len(summaryParts) > 0 {
		joiner := "|"
		if hasError {
			joiner = " "
		}
		summaryDisplay = ui.Colorize(fmt.Sprintf(" [%s]", ui.Join(summaryParts, joiner)), ui.ColorDarkGray)
	} else if errOld == nil && errNew == nil && oldVersion.Commit == newVersion.Commit {
		// If commits are the same, just show that there's no change.
		summaryDisplay = ui.Colorize(" [no changes]", ui.ColorDarkGray)
	}

	// --- 3. Combine and Return ---
	return fmt.Sprintf("[%s] %s -> %s%s", def.ShortName, oldVersionDisplay, newVersionDisplay, summaryDisplay)
}

// formatVersionDisplay creates the colored string for one side of the version arrow (->).
func formatVersionDisplay(v *git.Version, err error) string {
	if err != nil {
		return ui.Colorize("N/A", ui.ColorDarkGray)
	}

	// Shorten commit hash for display
	commit := v.Commit
	if len(commit) > 7 {
		commit = commit[:7]
	}

	versionParts := []string{v.Short()}
	if v.Branch != "" {
		versionParts = append(versionParts, v.Branch)
	}
	if commit != "" {
		versionParts = append(versionParts, commit)
	}

	return ui.Join(versionParts, ".")
}
