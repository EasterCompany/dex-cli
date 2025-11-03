package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/git"
	"github.com/EasterCompany/dex-cli/ui"
)

// Update manages the dex-cli update process
func Update(args []string) error {
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

	log("Updating to latest version...")
	ui.PrintInfo("Switching to main branch and pulling latest changes...")

	if err := git.SwitchBranch(sourcePath, "main"); err != nil {
		return err
	}
	log("Switched to branch 'main'")

	if err := git.Pull(sourcePath); err != nil {
		return err
	}
	log("Pulled latest changes")

	ui.PrintInfo("Waiting for 2 seconds to show pull logs...")
	time.Sleep(2 * time.Second)

	log("Building and installing `dex-cli` from source using Makefile...")
	ui.PrintInfo("Building and installing new version...")
	installCmd := exec.Command("make", "install")
	installCmd.Dir = sourcePath
	installCmd.Stdout = os.Stdout
	installCmd.Stderr = os.Stderr
	if err := installCmd.Run(); err != nil {
		log(fmt.Sprintf("make install failed: %v", err))
		return fmt.Errorf("make install failed: %w", err)
	}

	newVersion, _ := exec.Command("dex", "version").Output()
	log(fmt.Sprintf("Update complete. New version: %s", newVersion))
	ui.PrintInfo(fmt.Sprintf("Update complete. New version: %s", newVersion))

	return nil
}
