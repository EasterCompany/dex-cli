package cmd

import (
	"fmt"

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
	default:
		fmt.Printf("Unknown system command: %s\n", args[0])
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  dex system         # Show system info")
		fmt.Println("  dex system scan    # Re-scan hardware/software")
		fmt.Println("  dex system validate # Check for missing packages")
		return fmt.Errorf("unknown command")
	}
}

// systemInfo shows current system configuration
func systemInfo() error {
	sys, err := config.LoadSystemConfig()
	if err != nil {
		return fmt.Errorf("failed to load system config: %w", err)
	}

	config.PrintSystemInfo(sys)
	return nil
}

// systemScan re-scans hardware and updates system.json
func systemScan() error {
	fmt.Println("Scanning system...")
	fmt.Println()

	sys, err := config.IntrospectSystem()
	if err != nil {
		return fmt.Errorf("failed to scan system: %w", err)
	}

	if err := config.SaveSystemConfig(sys); err != nil {
		return fmt.Errorf("failed to save system config: %w", err)
	}

	fmt.Println("✓ System scan complete")
	fmt.Println()

	config.PrintSystemInfo(sys)
	return nil
}

// systemValidate checks for missing required packages
func systemValidate() error {
	sys, err := config.LoadSystemConfig()
	if err != nil {
		return fmt.Errorf("failed to load system config: %w", err)
	}

	fmt.Println("Validating system packages...")
	fmt.Println()

	missing := []config.Package{}
	for _, pkg := range sys.Packages {
		if pkg.Required && !pkg.Installed {
			missing = append(missing, pkg)
		}
	}

	if len(missing) == 0 {
		fmt.Println("✓ All required packages installed")
		return nil
	}

	fmt.Printf("✗ Missing %d required package(s):\n", len(missing))
	fmt.Println()

	for _, pkg := range missing {
		fmt.Printf("  %s (>= %s)\n", pkg.Name, pkg.MinVersion)
		if pkg.InstallCommand != "" {
			fmt.Printf("    Install: %s\n", pkg.InstallCommand)
		}
	}

	return fmt.Errorf("missing required packages")
}
