package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/git"
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

	// Check if the source directory exists
	sourcePath, err := config.ExpandPath("~/EasterCompany/dex-cli")
	if err != nil {
		return err
	}
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		log("Update command is disabled because the source code is not found.")
		log("If you want to enable updates, please install from source:")
		log("curl -fsSL https://raw.githubusercontent.com/eastercompany/dex-cli/main/install.sh | bash")
		return nil
	}

	branch := "main"
	commit := "latest"

	if len(args) > 0 {
		if strings.HasPrefix(args[0], "@") {
			commit = args[0][1:]
		} else {
			branch = args[0]
			if len(args) > 1 && strings.HasPrefix(args[1], "@") {
				commit = args[1][1:]
			}
		}
	}

	log(fmt.Sprintf("Updating to %s@%s...", branch, commit))

	// Git Source Control
	log("Updating git repository...")
	if err := git.SwitchBranch(sourcePath, branch); err != nil {
		return err
	}
	log(fmt.Sprintf("Switched to branch '%s'", branch))

	if commit == "latest" {
		if err := git.Pull(sourcePath); err != nil {
			return err
		}
		log("Pulled latest changes")
	} else {
		if err := git.CheckoutCommit(sourcePath, commit); err != nil {
			return err
		}
		log(fmt.Sprintf("Checked out commit '%s'", commit))
	}

	// Build & Install
	log("Building `dex-cli` from source...")
	buildCmd := exec.Command("go", "build", "-o", "dex-cli", ".")
	buildCmd.Dir = sourcePath
	if output, err := buildCmd.CombinedOutput(); err != nil {
		log(string(output))
		return fmt.Errorf("build failed: %w", err)
	}

	log("Installing `dex-cli` to `~/Dexter/bin`...")
	sourcePathBin := filepath.Join(sourcePath, "dex-cli")
	destPath, err := config.ExpandPath("~/Dexter/bin/dex")
	if err != nil {
		return err
	}
	if err := os.Rename(sourcePathBin, destPath); err != nil {
		return fmt.Errorf("install failed: %w", err)
	}

	newVersion, _ := exec.Command("dex", "version").Output()
	log(fmt.Sprintf("Update complete. New version: %s", strings.TrimSpace(string(newVersion))))

	return nil
}
