package cmd

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/git"
)

type pullResult struct {
	serviceName string
	status      string // "success", "skipped", "error"
	message     string
	icon        string
}

func handleCacheService() pullResult {
	// Check for redis-cli or valkey-cli
	cliPath, err := exec.LookPath("redis-cli")
	if err != nil {
		cliPath, err = exec.LookPath("valkey-cli")
		if err != nil {
			return pullResult{"cache", "error", "redis-cli or valkey-cli not found", "✗"}
		}
	}

	// Check service status
	cmd := exec.Command("systemctl", "is-active", "redis")
	out, err := cmd.Output()
	if err != nil || strings.TrimSpace(string(out)) != "active" {
		cmd = exec.Command("systemctl", "is-active", "valkey")
		out, err = cmd.Output()
		if err != nil || strings.TrimSpace(string(out)) != "active" {
			return pullResult{"cache", "error", "service not active", "✗"}
		}
	}

	// Ping the service
	cmd = exec.Command(cliPath, "ping")
	out, err = cmd.Output()
	if err != nil || strings.TrimSpace(string(out)) != "PONG" {
		return pullResult{"cache", "error", "ping failed", "✗"}
	}

	return pullResult{"cache", "success", "responding", "✓"}
}

// Pull synchronizes all Dexter services from their Git repositories
func Pull() error {
	// Ensure directory structure
	if err := config.EnsureDirectoryStructure(); err != nil {
		return fmt.Errorf("failed to ensure directory structure: %w", err)
	}

	// Validate packages
	if sys, err := config.LoadSystemConfig(); err == nil {
		missing := []string{}
		for _, pkg := range sys.Packages {
			if pkg.Required && !pkg.Installed {
				missing = append(missing, pkg.Name)
			}
		}
		if len(missing) > 0 {
			warningStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("220")).
				Padding(0, 1)
			fmt.Println(warningStyle.Render(fmt.Sprintf("⚠ Missing packages: %v", missing)))
			fmt.Println()
		}
	}

	// Load service map
	serviceMap, err := config.LoadServiceMap()
	if err != nil {
		return fmt.Errorf("failed to load service map: %w", err)
	}

	// Count total services
	totalServices := 0
	for _, services := range serviceMap.Services {
		totalServices += len(services)
	}

	// Process each service and collect results
	var results []pullResult
	successCount := 0
	skipCount := 0
	errorCount := 0

	for serviceType, services := range serviceMap.Services {
		if len(services) == 0 {
			continue
		}

		// Get service type label
		var serviceTypeLabel string
		for _, st := range serviceMap.ServiceTypes {
			if st.Type == serviceType {
				serviceTypeLabel = st.Label
				break
			}
		}
		if serviceTypeLabel == "" {
			serviceTypeLabel = serviceType
		}

		if serviceType == "os" {
			for _, service := range services {
				if service.ID == "cache" {
					result := handleCacheService()
															_ = append(results, result)
															renderServiceResult(result)
					switch result.status {
					case "success":
						successCount++
					case "error":
						errorCount++
					}
					continue
				}
			}
		} else {
			for _, service := range services {
				// Skip services without a repo URL
				if service.Repo == "" {
					                    															result := pullResult{service.ID, "skipped", "no repository configured", "○"}
					                    															_ = append(results, result)
					                    															renderServiceResult(result)
					                    															skipCount++
					continue
				}

				// Expand source path
				sourcePath, err := config.ExpandPath(service.Source)
				if err != nil {
					result := pullResult{service.ID, "error", "path expansion failed", "✗"}
					_ = append(results, result)
					renderServiceResult(result)
					errorCount++
					continue
				}

				// Check repository status
				status, err := git.CheckRepoStatus(sourcePath)
				if err != nil {
					result := pullResult{service.ID, "error", err.Error(), "✗"}
					results = append(results, result)
					renderServiceResult(result)
					errorCount++
					continue
				}

				// Handle non-existent repository (clone)
				if !status.Exists {
					if err := git.Clone(service.Repo, sourcePath); err != nil {
						result := pullResult{service.ID, "error", "clone failed", "✗"}
						results = append(results, result)
						renderServiceResult(result)
						errorCount++
						continue
					}

					result := pullResult{service.ID, "success", "cloned", "✓"}
					results = append(results, result)
					renderServiceResult(result)
					successCount++
					continue
				}

				// Repository exists, check if we can pull
				if !status.IsClean {
					result := pullResult{service.ID, "skipped", "uncommitted changes", "⊘"}
					results = append(results, result)
					renderServiceResult(result)
					skipCount++
					continue
				}

				if status.AheadOfRemote {
					result := pullResult{service.ID, "skipped", "unpushed commits", "↑"}
					results = append(results, result)
					renderServiceResult(result)
					skipCount++
					continue
				}

				if !status.BehindRemote {
					result := pullResult{service.ID, "success", "up to date", "✓"}
					results = append(results, result)
					renderServiceResult(result)
					successCount++
					continue
				}

				// Safe to pull
				if err := git.Pull(sourcePath); err != nil {
					result := pullResult{service.ID, "error", "pull failed", "✗"}
					results = append(results, result)
					renderServiceResult(result)
					errorCount++
					continue
				}

				result := pullResult{service.ID, "success", "updated", "✓"}
				results = append(results, result)
				renderServiceResult(result)
				successCount++
			}
		}
	}

	// Summary
	fmt.Println()
	renderSummary(totalServices, successCount, skipCount, errorCount)

	if errorCount > 0 {
		return fmt.Errorf("completed with %d error(s)", errorCount)
	}

	return nil
}

func renderServiceResult(result pullResult) {
	var style lipgloss.Style
	var statusText string

	switch result.status {
	case "success":
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
		statusText = result.message
	case "skipped":
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
		statusText = result.message
	case "error":
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
		statusText = result.message
	}

	serviceStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		Width(25).
		Padding(0, 1)

	iconStyle := lipgloss.NewStyle().
		Foreground(style.GetForeground()).
		Bold(true)

	line := fmt.Sprintf("%s %s %s",
		iconStyle.Render(result.icon),
		serviceStyle.Render(result.serviceName),
		style.Render(statusText))

	fmt.Println(line)
}

func renderSummary(total, success, skipped, errors int) {
	summaryBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("99")).
		Padding(1, 2).
		Width(50)

	var summaryText strings.Builder

	totalStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	summaryText.WriteString(totalStyle.Render(fmt.Sprintf("Total services:  %d", total)))
	summaryText.WriteString("\n")

	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	summaryText.WriteString(successStyle.Render(fmt.Sprintf("✓ Success:       %d", success)))
	summaryText.WriteString("\n")

	if skipped > 0 {
		skipStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
		summaryText.WriteString(skipStyle.Render(fmt.Sprintf("⊘ Skipped:       %d", skipped)))
		summaryText.WriteString("\n")
	}

	if errors > 0 {
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
		summaryText.WriteString(errorStyle.Render(fmt.Sprintf("✗ Errors:        %d", errors)))
	} else {
		allGoodStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("42")).
			Bold(true).
			MarginTop(1)
		summaryText.WriteString("\n")
		summaryText.WriteString(allGoodStyle.Render("All services synchronized!"))
	}

	fmt.Println(summaryBox.Render(summaryText.String()))
}
