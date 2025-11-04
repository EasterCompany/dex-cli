package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/EasterCompany/dex-cli/config"
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

	// ---
	// 1. Get current dex-cli version info (BEFORE update)
	// ---
	currentVersionStr, currentSize := getDexCliInfo()
	log("Updating to latest version...")
	ui.PrintSection("Downloading & Building")

	allServices := config.GetAllServices()
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
	installCmd := exec.Command("make", "install")
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
	}

	// ---
	// 4. Final Summary
	// ---
	newVersionStr, newSize := getDexCliInfo()
	latestVersion := ui.FetchLatestVersion()

	log(fmt.Sprintf("Update complete. New version: %s", newVersionStr))
	fmt.Println()
	ui.PrintSection("Complete")
	ui.PrintSuccess("All services are up to date.")
	fmt.Println()

	// ---
	// FIX: Pass currentSize and newSize to the function
	// ---
	ui.PrintVersionComparison(currentVersionStr, newVersionStr, latestVersion, buildYear, currentSize, newSize, 0, 0)

	return nil
}

// getDexCliInfo fetches the current version and binary size of dex-cli.
func getDexCliInfo() (version string, size int64) {
	currentVersion, _ := exec.Command("dex", "version").Output()
	version = strings.TrimSpace(string(currentVersion))

	currentBinaryPath, _ := exec.LookPath("dex")
	if currentBinaryPath != "" {
		if stat, err := os.Stat(currentBinaryPath); err == nil {
			size = stat.Size()
		}
	}
	return version, size
}
