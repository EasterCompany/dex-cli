package cmd

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/ui"
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
		fmt.Println(ui.RenderTitle("DEXTER SYSTEM"))
		fmt.Println()

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
	fmt.Println(ui.RenderTitle("SYSTEM SCAN"))
	fmt.Println()

	scanStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("111")).
		Italic(true)
	fmt.Println(scanStyle.Render("Scanning hardware and software..."))
	fmt.Println()

	sys, err := config.IntrospectSystem()
	if err != nil {
		return fmt.Errorf("failed to scan system: %w", err)
	}

	if err := config.SaveSystemConfig(sys); err != nil {
		return fmt.Errorf("failed to save system config: %w", err)
	}

	successStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("42")).
		Bold(true)
	fmt.Println(successStyle.Render("✓ System scan complete"))
	fmt.Println()

	renderSystemInfo(sys)
	return nil
}

// systemValidate checks for missing required packages
func systemValidate() error {
	fmt.Println(ui.RenderTitle("PACKAGE VALIDATION"))
	fmt.Println()

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
	fmt.Println(ui.RenderTitle("SYSTEM INFORMATION"))
	fmt.Println()

	// Create main info box
	infoBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("99")).
		Padding(1, 2).
		Width(70)

	var content strings.Builder

	// RAM
	ramGB := float64(sys.MemoryBytes) / (1024 * 1024 * 1024)

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("99"))

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		Width(12)

	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("111"))

	content.WriteString(headerStyle.Render("MEMORY"))
	content.WriteString("\n")
	content.WriteString(labelStyle.Render("Total RAM:"))
	content.WriteString(valueStyle.Render(fmt.Sprintf("  %.1f GB", ramGB)))
	content.WriteString("\n\n")

	// CPU
	content.WriteString(headerStyle.Render("PROCESSOR"))
	content.WriteString("\n")
	for _, cpu := range sys.CPU {
		cpuStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
		content.WriteString(cpuStyle.Render(cpu.Label))
		content.WriteString("\n")

		detailStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			Padding(0, 2)
		content.WriteString(detailStyle.Render(fmt.Sprintf("Cores: %d  •  Threads: %d", cpu.Count, cpu.Threads)))
		content.WriteString("\n")
	}
	content.WriteString("\n")

	// GPU
	if len(sys.GPU) > 0 {
		content.WriteString(headerStyle.Render("GRAPHICS"))
		content.WriteString("\n")
		for i, gpu := range sys.GPU {
			gpuStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
			content.WriteString(gpuStyle.Render(fmt.Sprintf("[%d] %s", i, gpu.Label)))
			content.WriteString("\n")

			if gpu.VRAM > 0 {
				vramGB := float64(gpu.VRAM) / (1024 * 1024 * 1024)
				detailStyle := lipgloss.NewStyle().
					Foreground(lipgloss.Color("245")).
					Padding(0, 4)
				content.WriteString(detailStyle.Render(fmt.Sprintf("VRAM: %.1f GB  •  CUDA: %d", vramGB, gpu.CUDA)))
				content.WriteString("\n")
			}
		}
		content.WriteString("\n")
	}

	// Storage
	if len(sys.Storage) > 0 {
		content.WriteString(headerStyle.Render("STORAGE"))
		content.WriteString("\n")
		for _, disk := range sys.Storage {
			diskStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("111")).
				Padding(0, 2)
			content.WriteString(diskStyle.Render(disk))
			content.WriteString("\n")
		}
		content.WriteString("\n")
	}

	// Packages
	content.WriteString(headerStyle.Render("PACKAGES"))
	content.WriteString("\n")

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
			Width(15).
			Padding(0, 1)

		versionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))

		line := fmt.Sprintf("%s %s %s",
			statusStyle.Render(icon),
			pkgNameStyle.Render(pkg.Name),
			versionStyle.Render(pkg.Version))

		if pkg.Required {
			requiredStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("220")).
				Italic(true)
			line += " " + requiredStyle.Render("(required)")
		}

		content.WriteString(line)
		content.WriteString("\n")
	}

	fmt.Println(infoBox.Render(content.String()))
}
