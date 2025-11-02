package cmd

import (
	"fmt"

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
	installed := []config.Package{}

	for _, pkg := range sys.Packages {
		if pkg.Required {
			if pkg.Installed {
				installed = append(installed, pkg)
			} else {
				missing = append(missing, pkg)
			}
		}
	}

	// Show installed packages
	if len(installed) > 0 {
		successStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("42")).
			Padding(0, 1).
			MarginBottom(1)
		fmt.Println(successStyle.Render("INSTALLED"))

		for _, pkg := range installed {
			pkgStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("252")).
				Padding(0, 1)
			versionStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("245"))

			fmt.Printf("  %s %s %s\n",
				lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Render("✓"),
				pkgStyle.Render(pkg.Name),
				versionStyle.Render(pkg.Version))
		}
		fmt.Println()
	}

	// Show missing packages
	if len(missing) > 0 {
		errorStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("196")).
			Padding(0, 1).
			MarginBottom(1)
		fmt.Println(errorStyle.Render("MISSING"))

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
	summaryBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("42")).
		Padding(1, 2)

	allGoodStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("42")).
		Bold(true)

	fmt.Println(summaryBox.Render(allGoodStyle.Render("✓ All required packages installed!")))
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
		fmt.Printf("%s  %s\n",
			headerStyle.Render("CPU:"),
			valueStyle.Render(cpu.Label))
		fmt.Printf("  %s\n",
			labelStyle.Render(fmt.Sprintf("Cores: %d  •  Threads: %d", cpu.Count, cpu.Threads)))
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
		totalSize := "Unknown"
		if len(sys.Storage) > 0 {
			// Try to parse the total from the storage list
			// For now, just use the first entry's size indication or "Multiple devices"
			if len(sys.Storage) == 1 {
				totalSize = "1 device"
			} else {
				totalSize = fmt.Sprintf("%d devices", len(sys.Storage))
			}
		}

		fmt.Printf("%s  %s\n",
			headerStyle.Render("STORAGE:"),
			valueStyle.Render(totalSize))
		for _, disk := range sys.Storage {
			fmt.Printf("  %s\n", labelStyle.Render(disk))
		}
	}

	// Blank line before PACKAGES
	fmt.Println()

	// Packages
	fmt.Printf("%s\n", headerStyle.Render("PACKAGES:"))

	for _, pkg := range sys.Packages {
		var statusStyle lipgloss.Style
		var icon string

		if pkg.Installed {
			statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
			icon = "✓"
		} else {
			statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
			icon = "✗"
		}

		pkgNameStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Width(18)

		versionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))

		line := fmt.Sprintf(" %s %s %s",
			statusStyle.Render(icon),
			pkgNameStyle.Render(pkg.Name),
			versionStyle.Render(pkg.Version))

		if pkg.Required {
			requiredStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("220"))
			line += " " + requiredStyle.Render("(required)")
		}

		fmt.Println(line)
	}
}
