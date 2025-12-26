package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/ui"
)

// System displays and manages system configuration
func System(args []string) error {
	outputJSON := false
	filteredArgs := []string{}
	for _, arg := range args {
		if arg == "--json" {
			outputJSON = true
			continue
		}
		if arg == "--help" || arg == "-h" {
			ui.PrintHeader("System Command Help")
			ui.PrintInfo("Usage: dex system [command] [args...]")
			fmt.Println()
			ui.PrintInfo("Commands:")
			ui.PrintInfo("  info       Show system info (default)")
			ui.PrintInfo("  scan       Re-scan hardware/software")
			ui.PrintInfo("  validate   Check for missing required packages")
			ui.PrintInfo("  install    [package] Install missing package(s)")
			ui.PrintInfo("  upgrade    [package] Upgrade installed package(s)")
			ui.PrintInfo("Flags:")
			ui.PrintInfo("  --json     Output system info as JSON")
			return nil
		}
		filteredArgs = append(filteredArgs, arg)
	}

	logFile, err := config.LogFile()
	if err != nil {
		return fmt.Errorf("failed to get log file: %w", err)
	}
	defer func() { _ = logFile.Close() }()

	log := func(message string) {
		_, _ = fmt.Fprintln(logFile, message)
	}

	log(fmt.Sprintf("System command called with args: %v", args))

	if len(filteredArgs) == 0 {
		return systemInfo(log, outputJSON)
	}

	switch filteredArgs[0] {
	case "info":
		return systemInfo(log, outputJSON)
	case "scan":
		return systemScan(log, outputJSON)
	case "validate":
		return systemValidate(log)
	case "install":
		return systemInstall(filteredArgs[1:], log)
	case "upgrade":
		return systemUpgrade(filteredArgs[1:], log)
	default:
		log(fmt.Sprintf("Unknown system subcommand: %s", filteredArgs[0]))
		fmt.Printf("Unknown command: %s\n", filteredArgs[0])
		return fmt.Errorf("unknown command")
	}
}

// systemInfo shows current system configuration
func systemInfo(log func(string), jsonOutput bool) error {
	log("Displaying system information.")
	sys, err := config.LoadSystemConfig()
	if err != nil {
		return fmt.Errorf("failed to load system config: %w", err)
	}

	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(sys)
	}

	table := ui.NewTable([]string{"Category", "Value"})

	// CPU
	for _, cpu := range sys.CPU {
		table.AddRow([]string{fmt.Sprintf("CPU (%s)", cpu.Label), fmt.Sprintf("Cores: %d, Threads: %d", cpu.Count, cpu.Threads)})
		if cpu.AvgGHz > 0 {
			table.AddRow([]string{"", fmt.Sprintf("Avg Clock: %.2f GHz", cpu.AvgGHz)})
		}
		if cpu.MaxGHz > 0 {
			table.AddRow([]string{"", fmt.Sprintf("Max Clock: %.2f GHz", cpu.MaxGHz)})
		}
	}

	// GPU
	if len(sys.GPU) > 0 {
		for i, gpu := range sys.GPU {
			gpuInfo := gpu.Label
			if gpu.VRAM > 0 {
				vramGB := float64(gpu.VRAM) / (1024 * 1024 * 1024)
				gpuInfo += fmt.Sprintf(", VRAM: %.1f GB", vramGB)
			}
			if gpu.CUDA > 0 {
				gpuInfo += fmt.Sprintf(", CUDA: %d", gpu.CUDA)
			}
			table.AddRow([]string{fmt.Sprintf("GPU %d", i), gpuInfo})
		}
	}

	// Memory
	ramGB := float64(sys.MemoryBytes) / (1024 * 1024 * 1024)
	table.AddRow([]string{"Memory", fmt.Sprintf("%.1f GB", ramGB)})

	// Storage
	if len(sys.Storage) > 0 {
		var totalSizeBytes int64
		for _, disk := range sys.Storage {
			totalSizeBytes += disk.Size
		}
		totalSizeGB := float64(totalSizeBytes) / (1024 * 1024 * 1024)
		table.AddRow([]string{"Storage", fmt.Sprintf("%.1f GB (%d devices)", totalSizeGB, len(sys.Storage))})

		for _, disk := range sys.Storage {
			sizeGB := float64(disk.Size) / (1024 * 1024 * 1024)
			var deviceInfo string
			if disk.MountPoint == "unmounted" || disk.MountPoint == "" {
				deviceInfo = fmt.Sprintf("%s: %.1f GB (unmounted)", disk.Device, sizeGB)
			} else {
				usedGB := float64(disk.Used) / (1024 * 1024 * 1024)
				deviceInfo = fmt.Sprintf("%s: %.1f GB / %.1f GB (%s)", disk.Device, usedGB, sizeGB, disk.MountPoint)
			}
			table.AddRow([]string{"", deviceInfo})
		}
	}

	// Packages
	table.AddRow([]string{"Packages", ""})
	missingPackages := []config.Package{}
	for _, pkg := range sys.Packages {
		if !pkg.Installed {
			missingPackages = append(missingPackages, pkg)
		}
	}

	totalCount := len(sys.Packages)
	missingCount := len(missingPackages)

	if missingCount > 0 {
		table.AddRow([]string{"", fmt.Sprintf("Found %d issues out of %d checks", missingCount, totalCount)})
		for _, pkg := range missingPackages {
			table.AddRow([]string{"  ✗ Missing", fmt.Sprintf("%s (>= %s)", pkg.Name, pkg.MinVersion)})
			if pkg.InstallCommand != "" {
				table.AddRow([]string{"", fmt.Sprintf("    Install: %s", pkg.InstallCommand)})
			}
		}
	} else {
		table.AddRow([]string{"", fmt.Sprintf("✓ %d checks passed", totalCount)})
	}

	table.Render()
	return nil
}

// systemScan re-scans hardware and updates system.json
func systemScan(log func(string), jsonOutput bool) error {
	log("Scanning system hardware and software.")
	sys, err := config.IntrospectSystem()
	if err != nil {
		return fmt.Errorf("failed to scan system: %w", err)
	}

	if err := config.SaveSystemConfig(sys); err != nil {
		return fmt.Errorf("failed to save system config: %w", err)
	}

	if err := config.EnsureConfigFiles(); err != nil {
		return fmt.Errorf("failed to ensure config files: %w", err)
	}

	if !jsonOutput {
		fmt.Println("✓ System scan complete")
	}
	log("System scan complete.")
	return systemInfo(log, jsonOutput)
}

// systemValidate checks for missing required packages
func systemValidate(log func(string)) error {
	log("Validating system packages.")
	sys, err := config.LoadSystemConfig()
	if err != nil {
		return fmt.Errorf("failed to load system config: %w", err)
	}

	missing := []config.Package{}
	requiredCount := 0

	for _, pkg := range sys.Packages {
		if pkg.Required {
			requiredCount++
			if !pkg.Installed {
				missing = append(missing, pkg)
			}
		}
	}

	missingCount := len(missing)

	if missingCount > 0 {
		fmt.Printf("Found %d issues out of %d required package checks\n", missingCount, requiredCount)
		table := ui.NewTable([]string{"Status", "Package", "Min Version", "Install Command"})
		for _, pkg := range missing {
			table.AddRow([]string{"✗ Missing", pkg.Name, pkg.MinVersion, pkg.InstallCommand})
		}
		table.Render()
		log(fmt.Sprintf("Missing %d required package(s).", missingCount))
		return fmt.Errorf("missing %d required package(s)", len(missing))
	}

	fmt.Printf("✓ %d required package checks passed\n", requiredCount)
	log("All required package checks passed.")
	return nil
}

// systemInstall installs missing packages
func systemInstall(args []string, log func(string)) error {
	log(fmt.Sprintf("Attempting to install packages: %v", args))
	sys, err := config.LoadSystemConfig()
	if err != nil {
		return fmt.Errorf("failed to load system config: %w", err)
	}

	if len(args) == 0 {
		// Install all missing packages
		missing := []config.Package{}
		for _, pkg := range sys.Packages {
			if !pkg.Installed {
				missing = append(missing, pkg)
			}
		}

		if len(missing) == 0 {
			fmt.Println("All packages are already installed.")
			log("All packages are already installed.")
			return nil
		}

		for _, pkg := range missing {
			if err := installPackage(pkg, log); err != nil {
				return err
			}
		}
		return nil
	}

	// Install specific package
	pkgName := args[0]
	for _, pkg := range sys.Packages {
		if pkg.Name == pkgName {
			if pkg.Installed {
				fmt.Printf("Package '%s' is already installed.\n", pkgName)
				log(fmt.Sprintf("Package '%s' is already installed.", pkgName))
				return nil
			}
			return installPackage(pkg, log)
		}
	}

	log(fmt.Sprintf("Package '%s' not found.", pkgName))
	return fmt.Errorf("package '%s' not found", pkgName)
}

func installPackage(pkg config.Package, log func(string)) error {
	if pkg.InstallCommand == "" {
		log(fmt.Sprintf("No install command found for package '%s'.", pkg.Name))
		return fmt.Errorf("no install command found for package '%s'", pkg.Name)
	}

	fmt.Printf("Installing '%s'...\n", pkg.Name)
	log(fmt.Sprintf("Installing '%s' by running: %s", pkg.Name, pkg.InstallCommand))

	cmd := exec.Command("bash", "-c", pkg.InstallCommand)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	err := cmd.Run()
	if err != nil {
		log(fmt.Sprintf("Failed to install package '%s': %v", pkg.Name, err))
		return fmt.Errorf("failed to install package '%s': %w", pkg.Name, err)
	}

	fmt.Printf("✓ Successfully installed '%s'.\n", pkg.Name)
	log(fmt.Sprintf("Successfully installed '%s'.", pkg.Name))
	return nil
}

// systemUpgrade upgrades installed packages
func systemUpgrade(args []string, log func(string)) error {
	log(fmt.Sprintf("Attempting to upgrade packages: %v", args))
	sys, err := config.LoadSystemConfig()
	if err != nil {
		return fmt.Errorf("failed to load system config: %w", err)
	}

	if len(args) == 0 {
		// Upgrade all installed packages
		for _, pkg := range sys.Packages {
			if pkg.Installed {
				if err := upgradePackage(pkg, log); err != nil {
					fmt.Printf("Failed to upgrade '%s': %v\n", pkg.Name, err)
					log(fmt.Sprintf("Failed to upgrade '%s': %v", pkg.Name, err))
				}
			}
		}
		return nil
	}

	// Upgrade specific package
	pkgName := args[0]
	for _, pkg := range sys.Packages {
		if pkg.Name == pkgName {
			if !pkg.Installed {
				log(fmt.Sprintf("Package '%s' is not installed.", pkgName))
				return fmt.Errorf("package '%s' is not installed", pkgName)
			}
			return upgradePackage(pkg, log)
		}
	}

	log(fmt.Sprintf("Package '%s' not found.", pkgName))
	return fmt.Errorf("package '%s' not found", pkgName)
}

func upgradePackage(pkg config.Package, log func(string)) error {
	if pkg.UpgradeCommand == "" {
		fmt.Printf("No upgrade command found for package '%s'. Skipping.\n", pkg.Name)
		log(fmt.Sprintf("No upgrade command found for package '%s'. Skipping.", pkg.Name))
		return nil
	}

	fmt.Printf("Upgrading '%s'...\n", pkg.Name)
	log(fmt.Sprintf("Upgrading '%s' by running: %s", pkg.Name, pkg.UpgradeCommand))

	cmd := exec.Command("bash", "-c", pkg.UpgradeCommand)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	err := cmd.Run()
	if err != nil {
		log(fmt.Sprintf("Failed to upgrade package '%s': %v", pkg.Name, err))
		return fmt.Errorf("failed to upgrade package '%s': %w", pkg.Name, err)
	}

	fmt.Printf("✓ Successfully upgraded '%s'.\n", pkg.Name)
	log(fmt.Sprintf("Successfully upgraded '%s'.", pkg.Name))
	return nil
}
