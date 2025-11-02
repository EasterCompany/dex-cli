package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/charmbracelet/lipgloss"

	"github.com/EasterCompany/dex-cli/config"
)

// System displays and manages system configuration
func System(args []string) error {
	if len(args) == 0 {
		return systemInfo()
	}

	switch args[0] {
	case "info":
		return systemInfo()
	case "scan":
		return systemScan()
	case "validate":
		return systemValidate()
	case "install":
		return systemInstall(args[1:])
	case "upgrade":
		return systemUpgrade(args[1:])
	default:
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
		fmt.Println(errorStyle.Render(fmt.Sprintf("Unknown command: %s", args[0])))
		fmt.Println()

		helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
		fmt.Println(helpStyle.Render("Available commands:"))

		cmdStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("111")).Padding(0, 2)
		fmt.Println(cmdStyle.Render("dex system         # Show system info"))
		fmt.Println(cmdStyle.Render("dex system scan    # Re-scan hardware/software"))
		fmt.Println(cmdStyle.Render("dex system validate # Check for missing packages"))
		fmt.Println(cmdStyle.Render("dex system install [package] # Install missing package(s)"))
		fmt.Println(cmdStyle.Render("dex system upgrade [package] # Upgrade installed package(s)"))

		return fmt.Errorf("unknown command")
	}
}

// systemInfo shows current system configuration
func systemInfo() error {
	sys, err := config.LoadSystemConfig()
	if err != nil {
		return fmt.Errorf("failed to load system config: %w", err)
	}

	renderSystemInfo(sys)
	return nil
}

// systemScan re-scans hardware and updates system.json
func systemScan() error {
	scanStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("111")).
		Italic(true)
	fmt.Println(scanStyle.Render("Scanning hardware and software..."))

	sys, err := config.IntrospectSystem()
	if err != nil {
		return fmt.Errorf("failed to scan system: %w", err)
	}

	if err := config.SaveSystemConfig(sys); err != nil {
		return fmt.Errorf("failed to save system config: %w", err)
	}

	successStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("42"))
	fmt.Println(successStyle.Render("✓ System scan complete"))
	fmt.Println()

	renderSystemInfo(sys)
	return nil
}

// systemValidate checks for missing required packages
func systemValidate() error {

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

	summaryStyle := lipgloss.NewStyle().Padding(0, 1)
	checksStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	issuesStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	okStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42"))

	if missingCount > 0 {
		summary := fmt.Sprintf("Found %s issues out of %s required package checks",
			issuesStyle.Render(fmt.Sprintf("%d", missingCount)),
			checksStyle.Render(fmt.Sprintf("%d", requiredCount)))
		fmt.Println(summaryStyle.Render(summary))

		for _, pkg := range missing {
			pkgStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("252")).
				Padding(0, 1)
			minVerStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("220"))

			fmt.Printf("  %s %s %s\n",
				lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("✗"),
				pkgStyle.Render(pkg.Name),
				minVerStyle.Render(fmt.Sprintf(">= %s", pkg.MinVersion)))

			if pkg.InstallCommand != "" {
				installStyle := lipgloss.NewStyle().
					Foreground(lipgloss.Color("111")).
					Padding(0, 4)
				fmt.Println(installStyle.Render(fmt.Sprintf("Install: %s", pkg.InstallCommand)))
			}
		}
		fmt.Println()

		return fmt.Errorf("missing %d required package(s)", len(missing))
	}

	// All good!
	summary := fmt.Sprintf("✓ %s required package checks passed", okStyle.Render(fmt.Sprintf("%d", requiredCount)))
	fmt.Println(summaryStyle.Render(summary))
	return nil
}

// systemInstall installs missing packages
func systemInstall(args []string) error {
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
			return nil
		}

		for _, pkg := range missing {
			if err := installPackage(pkg); err != nil {
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
				return nil
			}
			return installPackage(pkg)
		}
	}

	return fmt.Errorf("package '%s' not found", pkgName)
}

func installPackage(pkg config.Package) error {
	if pkg.InstallCommand == "" {
		return fmt.Errorf("no install command found for package '%s'", pkg.Name)
	}

	fmt.Printf("Installing '%s'...\n", pkg.Name)
	fmt.Printf("Running command: %s\n", pkg.InstallCommand)

	cmd := exec.Command("bash", "-c", pkg.InstallCommand)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to install package '%s': %w", pkg.Name, err)
	}

	fmt.Printf("✓ Successfully installed '%s'.\n", pkg.Name)
	return nil
}

// systemUpgrade upgrades installed packages
func systemUpgrade(args []string) error {
	sys, err := config.LoadSystemConfig()
	if err != nil {
		return fmt.Errorf("failed to load system config: %w", err)
	}

	if len(args) == 0 {
		// Upgrade all installed packages
		for _, pkg := range sys.Packages {
			if pkg.Installed {
				if err := upgradePackage(pkg); err != nil {
					fmt.Printf("Failed to upgrade '%s': %v\n", pkg.Name, err)
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
				return fmt.Errorf("package '%s' is not installed", pkgName)
			}
			return upgradePackage(pkg)
		}
	}

	return fmt.Errorf("package '%s' not found", pkgName)
}

func upgradePackage(pkg config.Package) error {
	if pkg.UpgradeCommand == "" {
		fmt.Printf("No upgrade command found for package '%s'. Skipping.\n", pkg.Name)
		return nil
	}

	fmt.Printf("Upgrading '%s'...\n", pkg.Name)
	fmt.Printf("Running command: %s\n", pkg.UpgradeCommand)

	cmd := exec.Command("bash", "-c", pkg.UpgradeCommand)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to upgrade package '%s': %w", pkg.Name, err)
	}

	fmt.Printf("✓ Successfully upgraded '%s'.\n", pkg.Name)
	return nil
}

func renderSystemInfo(sys *config.SystemConfig) {
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("99"))

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))

	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("111"))

	// CPU
	for _, cpu := range sys.CPU {
		fmt.Printf("%s\n", headerStyle.Render(fmt.Sprintf("CPU (%s):", cpu.Label)))
		fmt.Printf("  %s\n", labelStyle.Render(fmt.Sprintf("CORES: %d", cpu.Count)))
		fmt.Printf("  %s\n", labelStyle.Render(fmt.Sprintf("THREADS: %d", cpu.Threads)))
		if cpu.AvgGHz > 0 {
			fmt.Printf("  %s\n", labelStyle.Render(fmt.Sprintf("AVG: %.2f GHz", cpu.AvgGHz)))
		}
		if cpu.MaxGHz > 0 {
			fmt.Printf("  %s\n", labelStyle.Render(fmt.Sprintf("MAX: %.2f GHz", cpu.MaxGHz)))
		}
	}

	// GPU
	if len(sys.GPU) > 0 {
		for i, gpu := range sys.GPU {
			fmt.Printf("%s  %s\n",
				headerStyle.Render(fmt.Sprintf("GPU %d:", i)),
				valueStyle.Render(gpu.Label))
			if gpu.VRAM > 0 {
				vramGB := float64(gpu.VRAM) / (1024 * 1024 * 1024)
				fmt.Printf("  %s\n",
					labelStyle.Render(fmt.Sprintf("VRAM: %.1f GB  •  CUDA: %d", vramGB, gpu.CUDA)))
			}
		}
	}

	// Blank line before MEMORY/STORAGE group
	fmt.Println()

	// Memory
	ramGB := float64(sys.MemoryBytes) / (1024 * 1024 * 1024)
	fmt.Printf("%s  %s\n",
		headerStyle.Render("MEMORY:"),
		valueStyle.Render(fmt.Sprintf("%.1f GB", ramGB)))

	// Storage (no blank line between MEMORY and STORAGE)
	if len(sys.Storage) > 0 {
		// Calculate total storage
		var totalSizeBytes int64
		for _, disk := range sys.Storage {
			totalSizeBytes += disk.Size
		}
		totalSizeGB := float64(totalSizeBytes) / (1024 * 1024 * 1024)

		fmt.Printf("%s  %s\n",
			headerStyle.Render("STORAGE:"),
			valueStyle.Render(fmt.Sprintf("%.1f GB (%d devices)", totalSizeGB, len(sys.Storage))))

		for _, disk := range sys.Storage {
			sizeGB := float64(disk.Size) / (1024 * 1024 * 1024)

			var deviceInfo string
			if disk.MountPoint == "unmounted" || disk.MountPoint == "" {
				deviceInfo = fmt.Sprintf("%s: %.1f GB (unmounted)", disk.Device, sizeGB)
			} else {
				usedGB := float64(disk.Used) / (1024 * 1024 * 1024)
				deviceInfo = fmt.Sprintf("%s: %.1f GB / %.1f GB (%s)", disk.Device, usedGB, sizeGB, disk.MountPoint)
			}

			fmt.Printf("  %s\n", labelStyle.Render(deviceInfo))
		}
	}

	// Blank line before PACKAGES
	fmt.Println()

	// Packages
	fmt.Printf("%s\n", headerStyle.Render("PACKAGES:"))

	missingPackages := []config.Package{}
	for _, pkg := range sys.Packages {
		if !pkg.Installed {
			missingPackages = append(missingPackages, pkg)
		}
	}

	totalCount := len(sys.Packages)
	missingCount := len(missingPackages)

	summaryStyle := lipgloss.NewStyle().Padding(0, 1)
	checksStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	issuesStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	okStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42"))

	if missingCount > 0 {
		summary := fmt.Sprintf("Found %s issues out of %s checks",
			issuesStyle.Render(fmt.Sprintf("%d", missingCount)),
			checksStyle.Render(fmt.Sprintf("%d", totalCount)))
		fmt.Println(summaryStyle.Render(summary))

		for _, pkg := range missingPackages {
			pkgStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("252")).
				Padding(0, 1)
			minVerStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("220"))

			fmt.Printf("  %s %s %s\n",
				lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("✗"),
				pkgStyle.Render(pkg.Name),
				minVerStyle.Render(fmt.Sprintf(">= %s", pkg.MinVersion)))

			if pkg.InstallCommand != "" {
				installStyle := lipgloss.NewStyle().
					Foreground(lipgloss.Color("111")).
					Padding(0, 4)
				fmt.Println(installStyle.Render(fmt.Sprintf("Install: %s", pkg.InstallCommand)))
			}
		}
	} else {
		summary := fmt.Sprintf("✓ %s checks passed", okStyle.Render(fmt.Sprintf("%d", totalCount)))
		fmt.Println(summaryStyle.Render(summary))
	}
}
