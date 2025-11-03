package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/git"
	"github.com/EasterCompany/dex-cli/ui"
)

// Update manages the dex-cli update process
func Update(args []string, buildYear string) error {
	logFile, err := config.LogFile()
	if err != nil {
		return fmt.Errorf("failed to get log file: %w", err)
	}
	defer func() { _ = logFile.Close() }()

	log := func(message string) {
		_, _ = fmt.Fprintln(logFile, message)
	}

	sourcePath, err := config.ExpandPath("~/EasterCompany/dex-cli")
	if err != nil {
		return err
	}
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		ui.PrintError("Update command is disabled because the source code is not found.")
		ui.PrintInfo("To enable updates, please install from source:")
		ui.PrintInfo("curl -fsSL https://raw.githubusercontent.com/eastercompany/dex-cli/main/install.sh | bash")
		return nil
	}

	// Get current version and binary size before update
	currentVersion, _ := exec.Command("dex", "version").Output()
	currentVersionStr := strings.TrimSpace(string(currentVersion))

	currentBinaryPath, _ := exec.LookPath("dex")
	var currentSize int64
	if currentBinaryPath != "" {
		if stat, err := os.Stat(currentBinaryPath); err == nil {
			currentSize = stat.Size()
		}
	}

	log("Updating to latest version...")
	ui.PrintSection("Downloading")

	if err := git.SwitchBranch(sourcePath, "main"); err != nil {
		return err
	}
	log("Switched to branch 'main'")

	// Get commit hash before pull
	beforeCommit, _ := exec.Command("git", "-C", sourcePath, "rev-parse", "HEAD").Output()
	beforeCommitStr := strings.TrimSpace(string(beforeCommit))

	if err := git.Pull(sourcePath); err != nil {
		return err
	}
	log("Pulled latest changes")

	// Get commit hash after pull
	afterCommit, _ := exec.Command("git", "-C", sourcePath, "rev-parse", "HEAD").Output()
	afterCommitStr := strings.TrimSpace(string(afterCommit))

	// Get diff stats if commits changed
	var additions, deletions int
	if beforeCommitStr != afterCommitStr {
		diffOut, _ := exec.Command("git", "-C", sourcePath, "diff", "--shortstat", beforeCommitStr, afterCommitStr).Output()
		diffStr := string(diffOut)
		_, err = fmt.Sscanf(diffStr, "%*d files changed, %d insertions(+), %d deletions(-)", &additions, &deletions)
		if err != nil {
			log(fmt.Sprintf("Failed to parse diff string: %v", err))
		}
	}

	log("Building and installing `dex-cli` from source using Makefile...")
	ui.PrintSection("Building & Installing")
	installCmd := exec.Command("make", "install")
	installCmd.Dir = sourcePath
	installCmd.Stdout = os.Stdout
	installCmd.Stderr = os.Stderr
	if err := installCmd.Run(); err != nil {
		log(fmt.Sprintf("make install failed: %v", err))
		return fmt.Errorf("make install failed: %w", err)
	}

	// Get new version and binary size after update
	newVersion, _ := exec.Command("dex", "version").Output()
	newVersionStr := strings.TrimSpace(string(newVersion))

	newBinaryPath, _ := exec.LookPath("dex")
	var newSize int64
	if newBinaryPath != "" {
		if stat, err := os.Stat(newBinaryPath); err == nil {
			newSize = stat.Size()
		}
	}

	log(fmt.Sprintf("Update complete. New version: %s", newVersionStr))
	ui.PrintSection("Complete")

	// Fetch latest version from easter.company (cached)
	latestVersion := ui.FetchLatestVersion()
	if latestVersion == "" {
		log("Failed to fetch latest version from easter.company")
	}

	ui.PrintVersionComparison(currentVersionStr, newVersionStr, latestVersion, buildYear, currentSize, newSize, additions, deletions)

	return nil
}
