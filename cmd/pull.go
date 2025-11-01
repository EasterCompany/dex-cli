package cmd

import (
	"fmt"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/git"
)

// Pull synchronizes all Dexter services from their Git repositories
func Pull() error {
	fmt.Println("=== Dexter Pull Command ===")
	fmt.Println()

	// Ensure directory structure exists
	fmt.Println("Ensuring directory structure...")
	if err := config.EnsureDirectoryStructure(); err != nil {
		return fmt.Errorf("failed to ensure directory structure: %w", err)
	}
	fmt.Println("✓ Directory structure verified")
	fmt.Println()

	// Check system packages
	fmt.Println("Validating system packages...")
	if sys, err := config.LoadSystemConfig(); err == nil {
		missing := []string{}
		for _, pkg := range sys.Packages {
			if pkg.Required && !pkg.Installed {
				missing = append(missing, pkg.Name)
			}
		}
		if len(missing) > 0 {
			fmt.Printf("⚠ Missing packages: %v\n", missing)
			fmt.Println("  Run: dex system validate")
		} else {
			fmt.Println("✓ All required packages present")
		}
	}
	fmt.Println()

	// Load service map
	fmt.Println("Loading service map...")
	serviceMap, err := config.LoadServiceMap()
	if err != nil {
		return fmt.Errorf("failed to load service map: %w", err)
	}
	fmt.Printf("✓ Loaded %d service types\n", len(serviceMap.ServiceTypes))
	fmt.Println()

	// Count total services
	totalServices := 0
	for _, services := range serviceMap.Services {
		totalServices += len(services)
	}
	fmt.Printf("Found %d services to sync\n", totalServices)
	fmt.Println()

	// Process each service
	successCount := 0
	skipCount := 0
	errorCount := 0

	for serviceType, services := range serviceMap.Services {
		if len(services) == 0 {
			continue
		}

		fmt.Printf("--- %s Services ---\n", serviceType)

		for _, service := range services {
			// Skip services without a repo URL
			if service.Repo == "" {
				fmt.Printf("⊘ %s: No repository configured, skipping\n", service.ID)
				skipCount++
				continue
			}

			fmt.Printf("\n%s:\n", service.ID)

			// Expand source path
			sourcePath, err := config.ExpandPath(service.Source)
			if err != nil {
				fmt.Printf("  ✗ Failed to expand path: %v\n", err)
				errorCount++
				continue
			}

			// Check repository status
			status, err := git.CheckRepoStatus(sourcePath)
			if err != nil {
				fmt.Printf("  ✗ Error checking status: %v\n", err)
				errorCount++
				continue
			}

			// Handle non-existent repository (clone)
			if !status.Exists {
				fmt.Printf("  Repository does not exist at %s\n", sourcePath)

				if err := git.Clone(service.Repo, sourcePath); err != nil {
					fmt.Printf("  ✗ Clone failed: %v\n", err)
					errorCount++
					continue
				}

				fmt.Printf("  ✓ Successfully cloned\n")
				successCount++
				continue
			}

			// Repository exists, check if we can pull
			fmt.Printf("  Repository exists (branch: %s)\n", status.CurrentBranch)

			if !status.IsClean {
				fmt.Printf("  ⚠ Uncommitted changes detected, skipping pull for safety\n")
				fmt.Printf("  → Please commit or stash your changes manually\n")
				skipCount++
				continue
			}

			if status.AheadOfRemote {
				fmt.Printf("  ⚠ Local commits ahead of remote, skipping pull for safety\n")
				fmt.Printf("  → Please push your changes manually\n")
				skipCount++
				continue
			}

			if !status.BehindRemote {
				fmt.Printf("  ✓ Already up to date\n")
				successCount++
				continue
			}

			// Safe to pull
			fmt.Printf("  Updates available, pulling...\n")
			if err := git.Pull(sourcePath); err != nil {
				fmt.Printf("  ✗ Pull failed: %v\n", err)
				errorCount++
				continue
			}

			fmt.Printf("  ✓ Successfully updated\n")
			successCount++
		}

		fmt.Println()
	}

	// Summary
	fmt.Println("=== Summary ===")
	fmt.Printf("Total services: %d\n", totalServices)
	fmt.Printf("  ✓ Success: %d\n", successCount)
	fmt.Printf("  ⊘ Skipped: %d\n", skipCount)
	if errorCount > 0 {
		fmt.Printf("  ✗ Errors: %d\n", errorCount)
	}
	fmt.Println()

	if errorCount > 0 {
		return fmt.Errorf("completed with %d error(s)", errorCount)
	}

	fmt.Println("✓ Pull complete!")
	return nil
}
