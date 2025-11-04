package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/ui"
)

// Build compiles all services from their local source.
// It runs the full (format, lint, test, build, install) pipeline.
func Build(args []string) error {
	logFile, err := config.LogFile()
	if err != nil {
		return fmt.Errorf("failed to get log file: %w", err)
	}
	defer func() { _ = logFile.Close() }()

	log := func(message string) {
		_, _ = fmt.Fprintln(logFile, message)
	}

	if len(args) > 0 {
		return fmt.Errorf("build command takes no arguments")
	}

	log("Build command called...")
	ui.PrintSection("Building All Services from Local Source")

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
	// 1. Process dex-cli FIRST
	// ---
	ui.PrintInfo(ui.Colorize("--- Building dex-cli ---", ui.ColorCyan))
	// For dex-cli, we run 'make install' as it correctly handles all steps
	sourcePath, _ := config.ExpandPath(dexCliDef.Source)
	if !checkFileExists(sourcePath) {
		return fmt.Errorf("dex-cli source code not found at %s. Run 'dex add' to download & install it.", sourcePath)
	}

	installCmd := exec.Command("make", "install")
	installCmd.Dir = sourcePath
	installCmd.Stdout = os.Stdout
	installCmd.Stderr = os.Stderr

	if err := installCmd.Run(); err != nil {
		log(fmt.Sprintf("dex-cli 'make install' failed: %v", err))
		return fmt.Errorf("dex-cli 'make install' failed: %w", err)
	}

	// ---
	// 2. Process all OTHER services
	// ---
	servicesBuilt := 0
	for _, def := range allServices {
		// Skip os type and dex-cli (which we just did)
		if !def.IsManageable() || def.ShortName == "cli" {
			continue
		}

		// Check if source exists before trying to build
		sourcePath, _ := config.ExpandPath(def.Source)
		if !checkFileExists(sourcePath) {
			ui.PrintWarning(fmt.Sprintf("Skipping %s: source code not found at %s. Run 'dex add' to download & install it.", def.ShortName, sourcePath))
			continue
		}

		fmt.Println()
		ui.PrintInfo(ui.Colorize(fmt.Sprintf("--- Building %s ---", def.ShortName), ui.ColorCyan))
		log(fmt.Sprintf("Building %s from local source...", def.ShortName))

		// 1. Format
		if err := runServicePipelineStep(def, "format"); err != nil {
			return err // Stop-on-failure
		}

		// 2. Lint
		if err := runServicePipelineStep(def, "lint"); err != nil {
			return err // Stop-on-failure
		}

		// 3. Test
		if err := runServicePipelineStep(def, "test"); err != nil {
			return err // Stop-on-failure
		}

		// 4. Build
		if err := runServicePipelineStep(def, "build"); err != nil {
			return err // Stop-on-failure
		}

		// 5. Install
		if err := installSystemdService(def); err != nil {
			return err // Stop-on-failure
		}

		ui.PrintSuccess(fmt.Sprintf("Successfully built and installed %s!", def.ShortName))
		servicesBuilt++
	}

	fmt.Println()
	ui.PrintSection("Complete")
	ui.PrintSuccess(fmt.Sprintf("Successfully built and installed %d service(s).", servicesBuilt+1)) // +1 for dex-cli
	return nil
}
