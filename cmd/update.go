package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/git"
	"github.com/EasterCompany/dex-cli/ui"
	"github.com/EasterCompany/dex-cli/utils"
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
	// 1. Get initial versions and sizes
	// ---
	allServices := config.GetAllServices()
	oldVersions := make(map[string]string)
	oldSizes := make(map[string]int64)
	for _, s := range allServices {
		if s.IsBuildable() {
			oldVersions[s.ID] = utils.GetServiceVersion(s)
			oldSizes[s.ID] = utils.GetBinarySize(s)
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
	ui.PrintInfo(ui.Colorize(fmt.Sprintf("# Updating %s", dexCliDef.ShortName), ui.ColorCyan))
	if err := utils.GitUpdateService(dexCliDef); err != nil {
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
		ui.PrintWarning(fmt.Sprintf("Could not parse tag '%s', defaulting to 0.0.0. Error: %v", latestTag, err))
		major, minor, patch = 0, 0, 0
	}

	newVersion := fmt.Sprintf("%d.%d.%d", major, minor, patch+1)
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

		ui.PrintInfo(ui.Colorize(fmt.Sprintf("# Updating %s", def.ShortName), ui.ColorCyan))

		// 1. Download
		if err := utils.GitUpdateService(def); err != nil {
			return err // Stop-on-failure
		}

		// 2. Format
		if err := utils.RunServicePipelineStep(def, "format"); err != nil {
			return err // Stop-on-failure
		}

		// 3. Lint
		if err := utils.RunServicePipelineStep(def, "lint"); err != nil {
			return err // Stop-on-failure
		}

		// 4. Test
		if err := utils.RunServicePipelineStep(def, "test"); err != nil {
			return err // Stop-on-failure
		}

		// 5. Build
		if err := utils.RunServicePipelineStep(def, "build"); err != nil {
			return err // Stop-on-failure
		}

		// 6. Install
		if err := utils.InstallSystemdService(def); err != nil {
			return err // Stop-on-failure
		}

		ui.PrintSuccess(fmt.Sprintf("Successfully updated and installed %s!", def.ShortName))
		ui.PrintInfo(fmt.Sprintf("%s  Previous Version: %s%s", ui.ColorDarkGray, oldVersions[def.ID], ui.ColorReset))
		ui.PrintInfo(fmt.Sprintf("%s  Current Version:  %s%s", ui.ColorDarkGray, utils.GetFullServiceVersion(def), ui.ColorReset))
	}

	// ---
	// 4. Final Summary
	// ---
	log("Update complete.")
	fmt.Println()
	ui.PrintHeader("Complete")

	// Add a small delay to allow services to restart
	time.Sleep(2 * time.Second)

	// Get new versions and print changes, but ONLY for services in the service map
	configuredServices, err := utils.GetConfiguredServices()
	if err != nil {
		// Don't fail the whole command, just warn.
		ui.PrintWarning(fmt.Sprintf("Could not load configured services for final summary: %v", err))
	} else {
		for _, s := range configuredServices {
			if s.IsBuildable() {
				oldVersionStr := oldVersions[s.ID]
				newVersionStr := utils.GetServiceVersion(s)
				oldSize := oldSizes[s.ID]
				newSize := utils.GetBinarySize(s)
				ui.PrintInfo(utils.FormatSummaryLine(s, oldVersionStr, newVersionStr, oldSize, newSize))
			}
		}
	}

	fmt.Println() // Add a blank line for spacing
	ui.PrintSuccess("All services are up to date.")

	return nil
}
