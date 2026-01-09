package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/release"
	"github.com/EasterCompany/dex-cli/ui"
	"github.com/EasterCompany/dex-cli/utils"
)

const (
	DataJSONURL = "https://easter.company/bin/data.json"
	TempDir     = "/tmp/dex-update"
)

// Update performs different update strategies based on environment
func Update(args []string) error {
	// Wipe Redis to ensure a clean state
	if err := utils.WipeRedis(context.Background()); err != nil {
		ui.PrintWarning(fmt.Sprintf("Failed to wipe Redis: %v", err))
	}

	// Check if this is a developer environment
	isDev := isDeveloperEnvironment()

	if isDev {
		return updateDeveloper()
	}
	return updateUser()
}

// isDeveloperEnvironment checks if ~/EasterCompany exists
func isDeveloperEnvironment() bool {
	easterCompanyDir := fmt.Sprintf("%s/EasterCompany", os.Getenv("HOME"))
	_, err := os.Stat(easterCompanyDir)
	return err == nil
}

// updateDeveloper performs nuclear fresh install from source
// SCORCHED EARTH: Clone fresh → build via Makefile → install ALL binaries
func updateDeveloper() error {
	ui.PrintHeader("Developer Update - Nuclear Fresh Install")
	ui.PrintWarning("Cloning fresh source and rebuilding everything from scratch...")

	// Fetch data.json
	ui.PrintInfo("Fetching latest dev version from easter.company...")
	data, err := fetchReleaseData()
	if err != nil {
		return fmt.Errorf("failed to fetch release data: %w", err)
	}

	if data.Latest.Dev == "" {
		return fmt.Errorf("no dev version found in data.json")
	}

	ui.PrintInfo(fmt.Sprintf("Latest dev version: %s", data.Latest.Dev))

	// Get list of services to update
	services := config.GetAllServices()
	var buildableServices []config.ServiceDefinition
	for _, s := range services {
		if s.IsBuildable() && s.Type != "os" {
			buildableServices = append(buildableServices, s)
		}
	}

	ui.PrintInfo(fmt.Sprintf("Updating %d services from source...", len(buildableServices)))

	// Clean temp directory
	_ = os.RemoveAll(TempDir)
	if err := os.MkdirAll(TempDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(TempDir) }()

	// Process each service
	for _, service := range buildableServices {
		if err := updateServiceFromSource(service); err != nil {
			ui.PrintWarning(fmt.Sprintf("Failed to update %s: %v", service.ShortName, err))
			ui.PrintWarning("Continuing with other services...")
		}
	}

	ui.PrintSuccess("Update complete!")
	ui.PrintInfo("Run 'dex version' to verify")

	return nil
}

// updateServiceFromSource clones, builds via Makefile, and installs ALL binaries
func updateServiceFromSource(service config.ServiceDefinition) error {
	ui.PrintInfo(fmt.Sprintf("Updating %s...", service.ShortName))

	// Construct GitHub URL
	repoURL := fmt.Sprintf("https://github.com/EasterCompany/%s.git", service.ID)
	tempServiceDir := filepath.Join(TempDir, service.ID)

	// Clone fresh (depth 1 for speed)
	ui.PrintInfo(fmt.Sprintf("  Cloning %s...", service.ID))
	cloneCmd := exec.Command("git", "clone", "--depth", "1", repoURL, tempServiceDir)
	if output, err := cloneCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git clone failed:\n%s", string(output))
	}

	// Build via Makefile (source of truth for ALL binaries)
	ui.PrintInfo(fmt.Sprintf("  Building %s via Makefile...", service.ShortName))
	buildCmd := exec.Command("make", "build")
	buildCmd.Dir = tempServiceDir
	buildCmd.Env = append(os.Environ(), fmt.Sprintf("BIN_DIR=%s/Dexter/bin", os.Getenv("HOME")))
	if output, err := buildCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("make build failed:\n%s", string(output))
	}

	// Makefile's "make install" target handles copying ALL binaries to ~/Dexter/bin
	// So if the build succeeded, binaries are already installed!

	ui.PrintSuccess(fmt.Sprintf("  ✓ %s updated (all binaries installed)", service.ShortName))
	return nil
}

// updateUser downloads and installs binaries from easter.company
func updateUser() error {
	ui.PrintHeader("User Update - Download Latest Release")

	// Fetch data.json
	ui.PrintInfo("Checking for updates...")
	data, err := fetchReleaseData()
	if err != nil {
		return fmt.Errorf("failed to fetch release data: %w", err)
	}

	if data.Latest.User == "" {
		return fmt.Errorf("no user version found in data.json")
	}

	// Get current version
	currentVersion := RunningVersion
	latestVersion := data.Latest.User

	ui.PrintInfo(fmt.Sprintf("Current version: %s", currentVersion))
	ui.PrintInfo(fmt.Sprintf("Latest version:  %s", latestVersion))

	// Compare versions
	if currentVersion == latestVersion {
		ui.PrintSuccess("Already running the latest version!")
		return nil
	}

	ui.PrintInfo("Update available! Downloading binaries...")

	// Extract short version (e.g., "2.1.0" from full version string)
	shortVersion := extractShortVersion(latestVersion)
	if shortVersion == "" {
		return fmt.Errorf("failed to parse version: %s", latestVersion)
	}

	// Get release info to find all binaries
	releaseInfo, exists := data.Releases[shortVersion]
	if !exists {
		return fmt.Errorf("release %s not found in data.json", shortVersion)
	}

	// Download and install each binary from the release
	for serviceName, platforms := range releaseInfo.Binaries {
		// We only support linux-amd64
		binary, exists := platforms["linux-amd64"]
		if !exists {
			ui.PrintWarning(fmt.Sprintf("No linux-amd64 binary for %s", serviceName))
			continue
		}

		if err := downloadAndInstallBinary(serviceName, binary.Path, binary.Checksum); err != nil {
			ui.PrintWarning(fmt.Sprintf("Failed to update %s: %v", serviceName, err))
		}
	}

	ui.PrintSuccess("Update complete!")
	ui.PrintInfo("Run 'dex version' to verify")

	return nil
}

// downloadAndInstallBinary downloads and installs a single binary with checksum verification
func downloadAndInstallBinary(serviceName, binaryPath, expectedChecksum string) error {
	ui.PrintInfo(fmt.Sprintf("Downloading %s...", serviceName))

	// Construct download URL
	url := fmt.Sprintf("https://easter.company%s", binaryPath)

	// Download to temp file
	binaryName := filepath.Base(binaryPath)
	tempFile := filepath.Join(os.TempDir(), binaryName)
	if err := downloadFile(url, tempFile); err != nil {
		return err
	}
	defer func() { _ = os.Remove(tempFile) }()

	// Verify checksum
	actualChecksum, err := release.CalculateChecksum(tempFile)
	if err != nil {
		return fmt.Errorf("failed to calculate checksum: %w", err)
	}

	if actualChecksum != expectedChecksum {
		return fmt.Errorf("checksum mismatch (expected %s, got %s)", expectedChecksum, actualChecksum)
	}

	// Install to ~/Dexter/bin
	destPath := filepath.Join(os.Getenv("HOME"), "Dexter", "bin", binaryName)
	if err := copyFile(tempFile, destPath); err != nil {
		return err
	}

	// Make executable
	if err := os.Chmod(destPath, 0755); err != nil {
		return err
	}

	ui.PrintSuccess(fmt.Sprintf("  ✓ %s updated", binaryName))
	return nil
}

// fetchReleaseData fetches data.json from easter.company
func fetchReleaseData() (*release.ReleaseData, error) {
	resp, err := http.Get(DataJSONURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch data.json: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	var data release.ReleaseData
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to parse data.json: %w", err)
	}

	return &data, nil
}

// downloadFile downloads a file from a URL
func downloadFile(url, destination string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	out, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	_, err = io.Copy(out, resp.Body)
	return err
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, input, 0755)
}

// extractShortVersion extracts "X.Y.Z" from full version string
func extractShortVersion(fullVersion string) string {
	parts := strings.Split(fullVersion, ".")
	if len(parts) >= 3 {
		return strings.Join(parts[0:3], ".")
	}
	return ""
}
