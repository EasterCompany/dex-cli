package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"time"

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

	log("Updating to latest version...")

	// ---
	// 1. Get initial versions
	// ---
	allServices := config.GetAllServices()
	oldVersions := make(map[string]string)
	for _, s := range allServices {
		if s.IsBuildable() {
			oldVersions[s.ID] = getServiceVersion(s)
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
	log("Update complete.")
	fmt.Println()
	ui.PrintHeader("Complete")
	ui.PrintSuccess("All services are up to date.")

	// Add a small delay to allow services to restart
	time.Sleep(2 * time.Second)

	// Get new versions and print changes
	for _, s := range allServices {
		if s.IsBuildable() {
			newVersion := getServiceVersion(s)
			fmt.Printf("Service: %s, Old Version: %s, New Version: %s\n", s.ShortName, oldVersions[s.ID], newVersion)
			if oldVersions[s.ID] != newVersion {
				ui.PrintInfo(fmt.Sprintf("  %s version updated from %s to %s", s.ShortName, oldVersions[s.ID], newVersion))
			}
		}
	}

	return nil
}
