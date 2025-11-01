package cmd

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/git"
	"github.com/EasterCompany/dex-cli/ui"
)

func handleCacheService() error {
	// Check for redis-cli or valkey-cli
	cliPath, err := exec.LookPath("redis-cli")
	if err != nil {
		cliPath, err = exec.LookPath("valkey-cli")
		if err != nil {
			return fmt.Errorf("redis-cli or valkey-cli not found in PATH")
		}
	}
	ui.PrintInfo(fmt.Sprintf("Found cache CLI at: %s", cliPath))

	// Check service status
	cmd := exec.Command("systemctl", "is-active", "redis")
	out, err := cmd.Output()
	if err != nil || strings.TrimSpace(string(out)) != "active" {
		cmd = exec.Command("systemctl", "is-active", "valkey")
		out, err = cmd.Output()
		if err != nil || strings.TrimSpace(string(out)) != "active" {
			return fmt.Errorf("redis or valkey service is not active")
		}
	}
	ui.PrintSuccess("Cache service is active")

	// Ping the service
	cmd = exec.Command(cliPath, "ping")
	out, err = cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to ping cache service: %w", err)
	}

	if strings.TrimSpace(string(out)) != "PONG" {
		return fmt.Errorf("cache service did not respond with PONG")
	}

	ui.PrintSuccess("Cache service responded with PONG")
	return nil
}

// Pull synchronizes all Dexter services from their Git repositories
func Pull() error {
	ui.PrintTitle("DEXTER PULL COMMAND")

	ui.PrintSectionTitle("ENSURING DIRECTORY STRUCTURE")
	if err := config.EnsureDirectoryStructure(); err != nil {
		return fmt.Errorf("failed to ensure directory structure: %w", err)
	}
	ui.PrintSuccess("Directory structure verified")

	ui.PrintSectionTitle("VALIDATING SYSTEM PACKAGES")
	if sys, err := config.LoadSystemConfig(); err == nil {
		missing := []string{}
		for _, pkg := range sys.Packages {
			if pkg.Required && !pkg.Installed {
				missing = append(missing, pkg.Name)
			}
		}
		if len(missing) > 0 {
			ui.PrintWarning(fmt.Sprintf("Missing packages: %v", missing))
			ui.PrintInfo("Run: dex system validate")
		} else {
			ui.PrintSuccess("All required packages present")
		}
	}

	ui.PrintSectionTitle("LOADING SERVICE MAP")
	serviceMap, err := config.LoadServiceMap()
	if err != nil {
		return fmt.Errorf("failed to load service map: %w", err)
	}
	ui.PrintInfo(fmt.Sprintf("Loaded %d service types", len(serviceMap.ServiceTypes)))

	// Count total services
	totalServices := 0
	for _, services := range serviceMap.Services {
		totalServices += len(services)
	}
	ui.PrintInfo(fmt.Sprintf("Found %d services to sync", totalServices))

	// Process each service
	successCount := 0
	skipCount := 0
	errorCount := 0

	for serviceType, services := range serviceMap.Services {
		if len(services) == 0 {
			continue
		}

		if serviceType == "os" {
			for _, service := range services {
				ui.PrintSectionTitle(strings.ToUpper(service.ID))
				if service.ID == "cache" {
					if err := handleCacheService(); err != nil {
						ui.PrintError(fmt.Sprintf("Cache service check failed: %v", err))
						errorCount++
					} else {
						successCount++
					}
					continue
				}
				// Handle other special 'os' services here
			}
		} else {
			// Find the label for the serviceType
			var serviceTypeLabel string
			for _, st := range serviceMap.ServiceTypes {
				if st.Type == serviceType {
					serviceTypeLabel = st.Label
					break
				}
			}
			if serviceTypeLabel == "" {
				serviceTypeLabel = serviceType // Fallback if label not found
			}
			ui.PrintSectionTitle(strings.ToUpper(serviceTypeLabel))

			for _, service := range services {
				// Skip services without a repo URL
				if service.Repo == "" {
					ui.PrintWarning(fmt.Sprintf("%s: No repository configured, skipping", service.ID))
					skipCount++
					continue
				}

				ui.PrintInfo(service.ID)

				// Expand source path
				sourcePath, err := config.ExpandPath(service.Source)
				if err != nil {
					ui.PrintError(fmt.Sprintf("Failed to expand path: %v", err))
					errorCount++
					continue
				}

				// Check repository status
				status, err := git.CheckRepoStatus(sourcePath)
				if err != nil {
					ui.PrintError(fmt.Sprintf("Error checking status: %v", err))
					errorCount++
					continue
				}

				// Handle non-existent repository (clone)
				if !status.Exists {
					ui.PrintInfo(fmt.Sprintf("Repository does not exist at %s", sourcePath))

					if err := git.Clone(service.Repo, sourcePath); err != nil {
						ui.PrintError(fmt.Sprintf("Clone failed: %v", err))
						errorCount++
						continue
					}

					ui.PrintSuccess("Successfully cloned")
					successCount++
					continue
				}

				// Repository exists, check if we can pull
				ui.PrintInfo(fmt.Sprintf("Repository exists (branch: %s)", status.CurrentBranch))

				if !status.IsClean {
					ui.PrintWarning("Uncommitted changes detected, skipping pull for safety")
					ui.PrintInfo("Please commit or stash your changes manually")
					skipCount++
					continue
				}

				if status.AheadOfRemote {
					ui.PrintWarning("Local commits ahead of remote, skipping pull for safety")
					ui.PrintInfo("Please push your changes manually")
					skipCount++
					continue
				}

				if !status.BehindRemote {
					ui.PrintSuccess("Already up to date")
					successCount++
					continue
				}

				// Safe to pull
				ui.PrintInfo("Updates available, pulling...")
				if err := git.Pull(sourcePath); err != nil {
					ui.PrintError(fmt.Sprintf("Pull failed: %v", err))
					errorCount++
					continue
				}

				ui.PrintSuccess("Successfully updated")
				successCount++
			}
		}
	}

	ui.PrintSectionTitle("SUMMARY")
	ui.PrintInfo(fmt.Sprintf("Total services: %d", totalServices))
	ui.PrintSuccess(fmt.Sprintf("Success: %d", successCount))
	ui.PrintWarning(fmt.Sprintf("Skipped: %d", skipCount))
	if errorCount > 0 {
		ui.PrintError(fmt.Sprintf("Errors: %d", errorCount))
	}

	if errorCount > 0 {
		return fmt.Errorf("completed with %d error(s)", errorCount)
	}

	ui.PrintSuccess("PULL COMPLETE!")
	return nil
}
