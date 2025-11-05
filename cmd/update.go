package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/git"
	"github.com/EasterCompany/dex-cli/ui"
)

// Update manages the unified update process for dex-cli and all other services.
// It fetches, builds, and installs all services, one by one.
func Update(args []string, buildYear string) error {
	logFile, err := config.LogFile()
	if err != nil {
		return fmt.Errorf("failed to get log file: %w", err)
	}
	defer func() { _ = logFile.Close() }()

	log := func(message string) {
		_, _ = fmt.Fprintln(logFile, message)
	}

	log("Updating to latest version...")

	// ---
	// 1. Get initial versions
	// ---
	allServices := config.GetAllServices()
	oldVersions := make(map[string]string)
	for _, s := range allServices {
		if s.IsBuildable() {
			if s.ShortName == "cli" {
				// For the CLI, the "old" version is the one currently running.
				oldVersions[s.ID] = RunningVersion
			} else {
				oldVersions[s.ID] = getServiceVersion(s)
			}
		}
	}

	var dexCliDef config.ServiceDefinition
	// Find dex-cli definition
	for _, s := range allServices {
		if s.ShortName == "cli" {
			dexCliDef = s
			break
		}
	}

	// ---
	// 2. Process dex-cli FIRST (always)
	// ---
	ui.PrintInfo(ui.Colorize("# Updating dex-cli", ui.ColorCyan))
	if err := gitUpdateService(dexCliDef); err != nil {
		return fmt.Errorf("failed to update dex-cli source: %w", err)
	}

	// For dex-cli, we run 'make install' as it correctly handles all steps
	sourcePath, _ := config.ExpandPath(dexCliDef.Source)

	// Automatically bump the patch version for the build
	latestTag, err := git.GetLatestTag(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to get latest git tag for dex-cli: %w", err)
	}

	major, minor, patch, err := git.ParseVersionTag(latestTag)
	if err != nil {
		// Fallback to a default if the tag is malformed
		ui.PrintWarning(fmt.Sprintf("Could not parse tag '%s', defaulting to v0.0.0. Error: %v", latestTag, err))
		major, minor, patch = 0, 0, 0
	}

	newVersion := fmt.Sprintf("v%d.%d.%d", major, minor, patch+1)
	ui.PrintInfo(fmt.Sprintf("  Bumping version: %s -> %s", latestTag, newVersion))

	installCmd := exec.Command("make", "install", fmt.Sprintf("VERSION=%s", newVersion))
	installCmd.Dir = sourcePath
	installCmd.Stdout = os.Stdout
	installCmd.Stderr = os.Stderr

	if err := installCmd.Run(); err != nil {
		log(fmt.Sprintf("dex-cli 'make install' failed: %v", err))
		return fmt.Errorf("dex-cli 'make install' failed: %w", err)
	}

	// ---
	// 3. Process all OTHER services *that are in the service map*
	// ---
	serviceMap, err := config.LoadServiceMapConfig()
	if err != nil {
		return fmt.Errorf("failed to load service-map.json: %w", err)
	}

	// Create a quick lookup map of installed services
	installedServices := make(map[string]bool)
	for _, serviceList := range serviceMap.Services {
		for _, serviceEntry := range serviceList {
			installedServices[serviceEntry.ID] = true
		}
	}

	for _, def := range allServices {
		// Skip os type and dex-cli (which we just did)
		if !def.IsManageable() || def.ShortName == "cli" {
			continue
		}

		// *** Only update services that are in the service-map.json ***
		if _, isInstalled := installedServices[def.ID]; !isInstalled {
			log(fmt.Sprintf("Skipping %s (not in service-map.json)", def.ShortName))
			continue
		}

		fmt.Println()
		ui.PrintInfo(ui.Colorize(fmt.Sprintf("# Updating %s", def.ShortName), ui.ColorCyan))

		// 1. Download
		if err := gitUpdateService(def); err != nil {
			return err // Stop-on-failure
		}

		// 2. Format
		if err := runServicePipelineStep(def, "format"); err != nil {
			return err // Stop-on-failure
		}

		// 3. Lint
		if err := runServicePipelineStep(def, "lint"); err != nil {
			return err // Stop-on-failure
		}

		// 4. Test
		if err := runServicePipelineStep(def, "test"); err != nil {
			return err // Stop-on-failure
		}

		// 5. Build
		if err := runServicePipelineStep(def, "build"); err != nil {
			return err // Stop-on-failure
		}

		// 6. Install
		if err := installSystemdService(def); err != nil {
			return err // Stop-on-failure
		}

		ui.PrintSuccess(fmt.Sprintf("Successfully updated and installed %s!", def.ShortName))
		ui.PrintInfo(fmt.Sprintf("%s  Previous Version: %s%s", ui.ColorDarkGray, oldVersions[def.ID], ui.ColorReset))
		ui.PrintInfo(fmt.Sprintf("%s  Current Version:  %s%s", ui.ColorDarkGray, getServiceVersion(def), ui.ColorReset))
	}

	// ---
	// 4. Final Summary
	// ---
	log("Update complete.")
	fmt.Println()
	ui.PrintHeader("Complete")

	// Add a small delay to allow services to restart
	time.Sleep(2 * time.Second)

	// Get new versions and print changes
	for _, s := range allServices {
		if s.IsBuildable() {
			oldVersionStr := oldVersions[s.ID]
			newVersionStr := getServiceVersion(s)

			oldVersion, errOld := git.Parse(oldVersionStr)
			newVersion, errNew := git.Parse(newVersionStr)

			var oldVersionDisplay, newVersionDisplay string

			// Format old version
			if errOld != nil {
				oldVersionDisplay = ui.Colorize("N/A", ui.ColorDarkGray)
			} else {
				shortTag := oldVersion.Short()
				if errNew == nil && oldVersion.Compare(newVersion) < 0 {
					shortTag = ui.Colorize(shortTag, ui.ColorBrightRed)
				} else {
					shortTag = ui.Colorize(shortTag, ui.ColorReset) // White
				}

				var branchAndCommit string
				if oldVersion.Branch != "" && oldVersion.Commit != "" {
					branchAndCommit = fmt.Sprintf(".%s.%s", oldVersion.Branch, oldVersion.Commit)
					branchAndCommit = ui.Colorize(branchAndCommit, ui.ColorDarkGray)
				}
				oldVersionDisplay = fmt.Sprintf("%s%s", shortTag, branchAndCommit)
			}

			// Format new version
			if errNew != nil {
				newVersionDisplay = ui.Colorize("N/A", ui.ColorDarkGray)
			} else {
				shortTag := newVersion.Short()
				if errOld == nil && newVersion.Compare(oldVersion) > 0 {
					shortTag = ui.Colorize(shortTag, ui.ColorGreen)
				} else {
					shortTag = ui.Colorize(shortTag, ui.ColorReset) // White
				}

				var branchAndCommit string
				if newVersion.Branch != "" && newVersion.Commit != "" {
					branchAndCommit = fmt.Sprintf(".%s.%s", newVersion.Branch, newVersion.Commit)
					branchAndCommit = ui.Colorize(branchAndCommit, ui.ColorDarkGray)
				}
				newVersionDisplay = fmt.Sprintf("%s%s", shortTag, branchAndCommit)
			}

			greyV := ui.Colorize("v", ui.ColorDarkGray)
			ui.PrintInfo(fmt.Sprintf("[%s] %s %s -> %s", s.ShortName, greyV, oldVersionDisplay, newVersionDisplay))
		}
	}

	fmt.Println() // Add a blank line for spacing
	ui.PrintSuccess("All services are up to date.")

	return nil
}
