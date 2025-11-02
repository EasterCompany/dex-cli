package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/ui"
)

type validationIssue struct {
	severity string // "error", "warning", "info"
	category string
	message  string
	icon     string
}

// Config is the entry point for the config command
func Config(subcommand string) error {
	switch subcommand {
	case "validate":
		return ValidateConfig()
	default:
		return fmt.Errorf("unknown config subcommand: %s", subcommand)
	}
}

// ValidateConfig checks all configuration files for correctness
func ValidateConfig() error {
	var allIssues []validationIssue

	// Validate directory structure
	dirIssues := validateDirectoryStructure()
	allIssues = append(allIssues, dirIssues...)

	// Validate service-map.json
	serviceMapIssues := validateServiceMap()
	allIssues = append(allIssues, serviceMapIssues...)

	// Validate system.json
	systemIssues := validateSystemConfig()
	allIssues = append(allIssues, systemIssues...)

	// Validate options.json
	optionsIssues := validateOptions()
	allIssues = append(allIssues, optionsIssues...)

	// Render results
	renderValidationResults(allIssues)

	// Return error if there are any error-level issues
	errorCount := 0
	for _, issue := range allIssues {
		if issue.severity == "error" {
			errorCount++
		}
	}

	if errorCount > 0 {
		return fmt.Errorf("validation failed with %d error(s)", errorCount)
	}

	return nil
}

func validateDirectoryStructure() []validationIssue {
	var issues []validationIssue

	dexterPath, _ := config.ExpandPath(config.DexterRoot)
	easterCompanyPath, _ := config.ExpandPath(config.EasterCompanyRoot)

	// Check main directories
	if _, err := os.Stat(dexterPath); os.IsNotExist(err) {
		issues = append(issues, validationIssue{
			"error", "directory", "~/Dexter does not exist", "✗",
		})
	}

	if _, err := os.Stat(easterCompanyPath); os.IsNotExist(err) {
		issues = append(issues, validationIssue{
			"warning", "directory", "~/EasterCompany does not exist", "⚠",
		})
	}

	// Check required subdirectories
	for _, dir := range config.RequiredDexterDirs {
		dirPath := filepath.Join(dexterPath, dir)
		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
			issues = append(issues, validationIssue{
				"error", "directory", fmt.Sprintf("~/Dexter/%s does not exist", dir), "✗",
			})
		}
	}

	// Check config files exist
	configFiles := []string{"service-map.json", "system.json", "options.json"}
	for _, file := range configFiles {
		filePath := filepath.Join(dexterPath, "config", file)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			issues = append(issues, validationIssue{
				"error", "directory", fmt.Sprintf("%s does not exist", file), "✗",
			})
		}
	}

	return issues
}

func validateServiceMap() []validationIssue {
	var issues []validationIssue

	serviceMap, err := config.LoadServiceMap()
	if err != nil {
		issues = append(issues, validationIssue{
			"error", "service-map", fmt.Sprintf("Failed to load: %v", err), "✗",
		})
		return issues
	}

	// Validate service types
	if len(serviceMap.ServiceTypes) == 0 {
		issues = append(issues, validationIssue{
			"error", "service-map", "No service types defined", "✗",
		})
	}

	seenTypes := make(map[string]bool)
	for _, st := range serviceMap.ServiceTypes {
		if st.Type == "" {
			issues = append(issues, validationIssue{
				"error", "service-map", "Service type missing 'type' field", "✗",
			})
		}
		if seenTypes[st.Type] {
			issues = append(issues, validationIssue{
				"error", "service-map", fmt.Sprintf("Duplicate service type: %s", st.Type), "✗",
			})
		}
		seenTypes[st.Type] = true

		if st.Label == "" {
			issues = append(issues, validationIssue{
				"warning", "service-map", fmt.Sprintf("Service type '%s' missing label", st.Type), "⚠",
			})
		}
	}

	// Validate services
	seenIDs := make(map[string]bool)
	portUsage := make(map[int]string)

	for serviceType, services := range serviceMap.Services {
		// Check if service type exists
		if !seenTypes[serviceType] {
			issues = append(issues, validationIssue{
				"error", "service-map", fmt.Sprintf("Unknown service type: %s", serviceType), "✗",
			})
		}

		for _, service := range services {
			// Check for missing ID
			if service.ID == "" {
				issues = append(issues, validationIssue{
					"error", "service-map", fmt.Sprintf("Service in '%s' missing ID", serviceType), "✗",
				})
				continue
			}

			// Check for duplicate IDs
			if seenIDs[service.ID] {
				issues = append(issues, validationIssue{
					"error", "service-map", fmt.Sprintf("Duplicate service ID: %s", service.ID), "✗",
				})
			}
			seenIDs[service.ID] = true

			// Check for missing source/repo (except for os services)
			if serviceType != "os" {
				if service.Source == "" {
					issues = append(issues, validationIssue{
						"warning", "service-map", fmt.Sprintf("%s: missing source path", service.ID), "⚠",
					})
				} else {
					// Check if source path uses ~ notation
					if !strings.HasPrefix(service.Source, "~") {
						issues = append(issues, validationIssue{
							"warning", "service-map", fmt.Sprintf("%s: source path should use ~ notation", service.ID), "⚠",
						})
					}

					// Check if source directory exists
					sourcePath, _ := config.ExpandPath(service.Source)
					if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
						issues = append(issues, validationIssue{
							"info", "service-map", fmt.Sprintf("%s: source directory does not exist", service.ID), "ℹ",
						})
					}
				}

				if service.Repo == "" {
					issues = append(issues, validationIssue{
						"warning", "service-map", fmt.Sprintf("%s: missing repository URL", service.ID), "⚠",
					})
				} else if !strings.HasPrefix(service.Repo, "git@github.com:") {
					issues = append(issues, validationIssue{
						"info", "service-map", fmt.Sprintf("%s: not using SSH git URL", service.ID), "ℹ",
					})
				}
			}

			// Validate port assignments
			if service.Addr != "" {
				port := extractPort(service.Addr)
				if port > 0 {
					// Check if port is in correct range for service type
					for _, st := range serviceMap.ServiceTypes {
						if st.Type == serviceType && st.MinPort > 0 {
							if port < st.MinPort || port > st.MaxPort {
								issues = append(issues, validationIssue{
									"warning", "service-map",
									fmt.Sprintf("%s: port %d outside range %d-%d for type '%s'",
										service.ID, port, st.MinPort, st.MaxPort, serviceType), "⚠",
								})
							}
							break
						}
					}

					// Check for port conflicts
					if existingService, exists := portUsage[port]; exists {
						issues = append(issues, validationIssue{
							"error", "service-map",
							fmt.Sprintf("Port conflict: %s and %s both use port %d",
								service.ID, existingService, port), "✗",
						})
					} else {
						portUsage[port] = service.ID
					}
				}
			}

			// Check for placeholder/default values
			if strings.Contains(service.Addr, "example.com") || strings.Contains(service.Addr, "localhost") {
				issues = append(issues, validationIssue{
					"info", "service-map", fmt.Sprintf("%s: using placeholder address", service.ID), "ℹ",
				})
			}
		}
	}

	return issues
}

func validateSystemConfig() []validationIssue {
	var issues []validationIssue

	sys, err := config.LoadSystemConfig()
	if err != nil {
		issues = append(issues, validationIssue{
			"error", "system", fmt.Sprintf("Failed to load: %v", err), "✗",
		})
		return issues
	}

	// Check for missing required packages
	for _, pkg := range sys.Packages {
		if pkg.Required && !pkg.Installed {
			issues = append(issues, validationIssue{
				"error", "system", fmt.Sprintf("Required package not installed: %s", pkg.Name), "✗",
			})
		}
	}

	// Check for reasonable memory
	if sys.MemoryBytes < 8*1024*1024*1024 { // Less than 8GB
		issues = append(issues, validationIssue{
			"warning", "system", fmt.Sprintf("Low memory detected: %.1f GB", float64(sys.MemoryBytes)/(1024*1024*1024)), "⚠",
		})
	}

	// Check for GPU presence
	if len(sys.GPU) == 0 {
		issues = append(issues, validationIssue{
			"info", "system", "No GPU detected (some services may run slower)", "ℹ",
		})
	}

	return issues
}

func validateOptions() []validationIssue {
	var issues []validationIssue

	optionsPath, _ := config.ExpandPath(filepath.Join(config.DexterRoot, "config", "options.json"))
	data, err := os.ReadFile(optionsPath)
	if err != nil {
		issues = append(issues, validationIssue{
			"error", "options", fmt.Sprintf("Failed to load: %v", err), "✗",
		})
		return issues
	}

	var options map[string]interface{}
	if err := json.Unmarshal(data, &options); err != nil {
		issues = append(issues, validationIssue{
			"error", "options", fmt.Sprintf("Invalid JSON: %v", err), "✗",
		})
		return issues
	}

	// Check for hardcoded paths
	dataStr := string(data)
	if strings.Contains(dataStr, "/home/") {
		issues = append(issues, validationIssue{
			"warning", "options", "Contains hardcoded /home/ paths (should use ~ notation)", "⚠",
		})
	}

	// Check for placeholder values
	if strings.Contains(dataStr, "YOUR_") || strings.Contains(dataStr, "REPLACE_") {
		issues = append(issues, validationIssue{
			"warning", "options", "Contains placeholder values that need replacement", "⚠",
		})
	}

	// Check for sensitive data patterns
	if discord, ok := options["discord"].(map[string]interface{}); ok {
		if token, ok := discord["token"].(string); ok {
			if token == "" || token == "YOUR_TOKEN_HERE" {
				issues = append(issues, validationIssue{
					"error", "options", "Discord token is missing or placeholder", "✗",
				})
			} else if len(token) < 50 {
				issues = append(issues, validationIssue{
					"warning", "options", "Discord token appears invalid (too short)", "⚠",
				})
			}
		}
	}

	// Check python configuration
	if python, ok := options["python"].(map[string]interface{}); ok {
		if venv, ok := python["venv"].(string); ok {
			if !strings.HasPrefix(venv, "~") {
				issues = append(issues, validationIssue{
					"warning", "options", "Python venv path should use ~ notation", "⚠",
				})
			}
		}
	}

	return issues
}

func extractPort(addr string) int {
	re := regexp.MustCompile(`:(\d+)`)
	matches := re.FindStringSubmatch(addr)
	if len(matches) > 1 {
		var port int
		fmt.Sscanf(matches[1], "%d", &port)
		return port
	}
	return 0
}

func renderValidationResults(issues []validationIssue) {
	// Group issues by category
	categoryMap := make(map[string][]validationIssue)
	for _, issue := range issues {
		categoryMap[issue.category] = append(categoryMap[issue.category], issue)
	}

	// Render each category
	categories := []string{"directory", "service-map", "system", "options"}
	for _, category := range categories {
		if categoryIssues, exists := categoryMap[category]; exists {
			renderCategory(category, categoryIssues)
		}
	}

	// Summary
	fmt.Println()
	renderValidationSummary(issues)
}

func renderCategory(category string, issues []validationIssue) {
	categoryStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("99")).
		Padding(0, 1).
		MarginTop(1)

	fmt.Println(categoryStyle.Render(strings.ToUpper(category)))

	for _, issue := range issues {
		var style lipgloss.Style
		switch issue.severity {
		case "error":
			style = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
		case "warning":
			style = lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
		case "info":
			style = lipgloss.NewStyle().Foreground(lipgloss.Color("111"))
		}

		iconStyle := lipgloss.NewStyle().
			Foreground(style.GetForeground()).
			Bold(true)

		fmt.Printf("%s %s\n",
			iconStyle.Render(issue.icon),
			style.Render(issue.message))
	}
}

func renderValidationSummary(issues []validationIssue) {
	errorCount := 0
	warningCount := 0
	infoCount := 0

	for _, issue := range issues {
		switch issue.severity {
		case "error":
			errorCount++
		case "warning":
			warningCount++
		case "info":
			infoCount++
		}
	}

	summaryBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("99")).
		Padding(1, 2).
		Width(50)

	var summaryText strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("99"))
	summaryText.WriteString(titleStyle.Render("VALIDATION SUMMARY"))
	summaryText.WriteString("\n\n")

	totalStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	summaryText.WriteString(totalStyle.Render(fmt.Sprintf("Total issues:    %d", len(issues))))
	summaryText.WriteString("\n")

	if errorCount > 0 {
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
		summaryText.WriteString(errorStyle.Render(fmt.Sprintf("✗ Errors:        %d", errorCount)))
		summaryText.WriteString("\n")
	}

	if warningCount > 0 {
		warningStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
		summaryText.WriteString(warningStyle.Render(fmt.Sprintf("⚠ Warnings:      %d", warningCount)))
		summaryText.WriteString("\n")
	}

	if infoCount > 0 {
		infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("111"))
		summaryText.WriteString(infoStyle.Render(fmt.Sprintf("ℹ Info:          %d", infoCount)))
		summaryText.WriteString("\n")
	}

	if errorCount == 0 && warningCount == 0 {
		allGoodStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("42")).
			Bold(true).
			MarginTop(1)
		summaryText.WriteString("\n")
		summaryText.WriteString(allGoodStyle.Render("All configurations valid!"))
	} else if errorCount == 0 {
		okStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("220")).
			MarginTop(1)
		summaryText.WriteString("\n")
		summaryText.WriteString(okStyle.Render("No critical errors, but warnings exist"))
	}

	fmt.Println(summaryBox.Render(summaryText.String()))
}
