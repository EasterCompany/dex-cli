package release

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/git"
	"github.com/EasterCompany/dex-cli/ui"
)

const (
	EasterCompanyRepo = "~/EasterCompany/easter.company"
	DataJSONPath      = "bin/data.json"
	BinPath           = "bin"
)

// PublishRelease publishes binaries to easter.company for major/minor releases
// version is the FULL version string (e.g., 2.1.0.main.abc123.2025-11-27-09-30-45.linux-amd64.xyz789)
func PublishRelease(fullVersion, shortVersion, releaseType string, services []config.ServiceDefinition) error {
	// Expand easter.company path
	repoPath, err := config.ExpandPath(EasterCompanyRepo)
	if err != nil {
		return fmt.Errorf("failed to expand easter.company path: %w", err)
	}

	// Verify repo exists
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		return fmt.Errorf("easter.company repo not found at %s - run: git clone git@github.com:eastercompany/eastercompany.github.io.git ~/EasterCompany/easter.company", repoPath)
	}

	ui.PrintInfo(fmt.Sprintf("Publishing %s release %s to easter.company...", releaseType, shortVersion))

	// Load or create data.json
	dataPath := filepath.Join(repoPath, DataJSONPath)
	data, err := LoadReleaseData(dataPath)
	if err != nil {
		return fmt.Errorf("failed to load data.json: %w", err)
	}

	// Get current git commit
	_, commit := git.GetVersionInfo(repoPath)

	// Handle version cleanup based on type
	if releaseType == "minor" {
		// Extract major version (e.g., "2" from "2.1.0")
		majorVersion := strings.Split(shortVersion, ".")[0]

		// Remove all other minors from this major
		removed := data.RemoveMinorVersions(majorVersion)
		if len(removed) > 0 {
			ui.PrintInfo(fmt.Sprintf("Removing previous minor versions: %v", removed))

			// Delete the binary directories
			for _, oldVer := range removed {
				oldDir := filepath.Join(repoPath, BinPath, oldVer)
				if err := os.RemoveAll(oldDir); err != nil {
					ui.PrintWarning(fmt.Sprintf("Failed to remove %s: %v", oldDir, err))
				}
			}
		}
	}

	// Add the new release (use short version as key, but store full version in data)
	data.AddRelease(shortVersion, releaseType, commit)

	// Create version directory (use short version for directory name)
	versionDir := filepath.Join(repoPath, BinPath, shortVersion)
	if err := os.MkdirAll(versionDir, 0755); err != nil {
		return fmt.Errorf("failed to create version directory: %w", err)
	}

	// Copy binaries and update data
	platform := "linux-amd64" // We only support linux-amd64
	for _, service := range services {
		binPath := filepath.Join(os.Getenv("HOME"), "Dexter", "bin", getBinaryName(service))

		if _, err := os.Stat(binPath); os.IsNotExist(err) {
			ui.PrintWarning(fmt.Sprintf("Binary not found: %s", binPath))
			continue
		}

		// Copy binary to version directory
		destPath := filepath.Join(versionDir, filepath.Base(binPath))
		if err := copyFile(binPath, destPath); err != nil {
			return fmt.Errorf("failed to copy %s: %w", binPath, err)
		}

		// Add to data.json (using short version as key)
		if err := data.AddBinary(shortVersion, service.ShortName, platform, destPath); err != nil {
			return fmt.Errorf("failed to add binary to data: %w", err)
		}

		// Get full version string for this service
		serviceFullVersion := getServiceFullVersion(service)

		// Update service info (store full version strings)
		repo := fmt.Sprintf("github.com/EasterCompany/%s", service.ID)
		data.UpdateService(service.ShortName, serviceFullVersion, serviceFullVersion, repo)

		ui.PrintSuccess(fmt.Sprintf("Published %s", filepath.Base(binPath)))

		// Special case for CLI: Update latest binary for install script
		if service.ShortName == "cli" {
			latestDir := filepath.Join(repoPath, BinPath, "latest")
			if err := os.MkdirAll(latestDir, 0755); err != nil {
				ui.PrintWarning(fmt.Sprintf("Failed to create latest directory: %v", err))
			} else {
				latestPath := filepath.Join(latestDir, "dex")
				if err := copyFile(binPath, latestPath); err != nil {
					ui.PrintWarning(fmt.Sprintf("Failed to update latest binary: %v", err))
				} else {
					ui.PrintSuccess("Updated bin/latest/dex")
				}
			}
		}
	}

	// Update latest versions (store full version strings)
	// Get the full version for cli (our canonical version)
	cliFullVersion := ""
	for _, service := range services {
		if service.ShortName == "cli" {
			cliFullVersion = getServiceFullVersion(service)
			break
		}
	}
	data.Latest.User = cliFullVersion
	data.Latest.Dev = cliFullVersion

	// Save data.json
	if err := data.Save(dataPath); err != nil {
		return fmt.Errorf("failed to save data.json: %w", err)
	}

	ui.PrintSuccess(fmt.Sprintf("Updated %s", DataJSONPath))

	// Git commit and push
	if err := commitAndPush(repoPath, shortVersion, releaseType); err != nil {
		return fmt.Errorf("failed to commit and push: %w", err)
	}

	return nil
}

// getBinaryName returns the binary name for a service
func getBinaryName(service config.ServiceDefinition) string {
	if service.ShortName == "cli" {
		return "dex"
	}
	return service.ID
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	if err := os.WriteFile(dst, input, 0755); err != nil {
		return err
	}

	return nil
}

// getServiceFullVersion gets the full version string for a service binary
func getServiceFullVersion(service config.ServiceDefinition) string {
	binPath := filepath.Join(os.Getenv("HOME"), "Dexter", "bin", getBinaryName(service))

	// Run the binary with 'version' argument to get full version
	cmd := exec.Command(binPath, "version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "unknown"
	}

	// Parse output - the output is expected to be just the version string
	fullVersion := strings.TrimSpace(string(output))
	if fullVersion != "" {
		return fullVersion
	}

	return "unknown"
}

// commitAndPush commits and pushes changes to easter.company repo
func commitAndPush(repoPath, version, releaseType string) error {
	ui.PrintInfo("Committing and pushing to easter.company...")

	// Git add
	addCmd := exec.Command("git", "add", ".")
	addCmd.Dir = repoPath
	addCmd.Stdout = os.Stdout
	addCmd.Stderr = os.Stderr
	if err := addCmd.Run(); err != nil {
		return fmt.Errorf("git add failed: %w", err)
	}

	// Check for changes
	statusCmd := exec.Command("git", "status", "--porcelain")
	statusCmd.Dir = repoPath
	statusOutput, err := statusCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git status failed:\n%s", string(statusOutput))
	}

	if strings.TrimSpace(string(statusOutput)) == "" {
		ui.PrintInfo("No changes to commit")
		return nil
	}

	// Commit
	commitMsg := fmt.Sprintf("release: publish %s version %s", releaseType, version)
	commitCmd := exec.Command("git", "commit", "-m", commitMsg)
	commitCmd.Dir = repoPath
	commitCmd.Stdout = os.Stdout
	commitCmd.Stderr = os.Stderr
	if err := commitCmd.Run(); err != nil {
		return fmt.Errorf("git commit failed: %w", err)
	}

	// Push
	pushCmd := exec.Command("git", "push")
	pushCmd.Dir = repoPath
	pushCmd.Stdout = os.Stdout
	pushCmd.Stderr = os.Stderr
	if err := pushCmd.Run(); err != nil {
		return fmt.Errorf("git push failed: %w", err)
	}

	ui.PrintSuccess("Published to https://easter.company")
	return nil
}
