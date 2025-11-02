package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/git"
	"github.com/EasterCompany/dex-cli/ui"
)

// Update manages the dex-cli update process
func Update(args []string) error {
	target := "main"
	force := false

	if len(args) > 0 {
		if args[0] != "-y" {
			target = args[0]
		} else {
			force = true
		}
	}
	if len(args) > 1 {
		if args[1] == "-y" {
			force = true
		}
	}

	// Get current version information
	fmt.Println(ui.RenderSectionTitle("Current Version"))
	dexCliPath, err := config.ExpandPath("~/EasterCompany/dex-cli")
	if err != nil {
		return err
	}
	installedVersion, _ := exec.Command("dex", "version").Output()
	localBranch, localCommit := git.GetVersionInfo(dexCliPath)
	repoStatus, err := git.CheckRepoStatus(dexCliPath)
	if err != nil {
		return err
	}

	ui.PrintInfo(fmt.Sprintf("Current `~/Dexter/bin/dex` version: %s", strings.TrimSpace(string(installedVersion))))
	ui.PrintInfo(fmt.Sprintf("Current `~/EasterCompany/dex-cli` version: %s @ %s", localBranch, localCommit))
	if repoStatus.HasUncommitted {
		ui.PrintWarning("Local changes detected")
	}
	if repoStatus.HasUnpushed {
		ui.PrintWarning("Local commits are not pushed")
	}
	if repoStatus.BehindRemote {
		ui.PrintInfo("Updates are available on the remote repository")
	}

	// Handle prompts
	if !force {
		if target == "local" {
			if repoStatus.HasUncommitted {
				// no prompt for local changes
			} else if repoStatus.BehindRemote {
				fmt.Print("There are changes on the remote repository, would you like to continue anyway? (Y/n) ")
				reader := bufio.NewReader(os.Stdin)
				input, _ := reader.ReadString('\n')
				if strings.TrimSpace(input) != "Y" && strings.TrimSpace(input) != "y" {
					return nil
				}
			} else {
				fmt.Print("There are no local changes, would you like to continue anyway? (Y/n) ")
				reader := bufio.NewReader(os.Stdin)
				input, _ := reader.ReadString('\n')
				if strings.TrimSpace(input) != "Y" && strings.TrimSpace(input) != "y" {
					return nil
				}
			}
		} else {
			if repoStatus.HasUncommitted || repoStatus.HasUnpushed {
				fmt.Print("There are local changes, would you like to continue anyway? (Y/n) ")
				reader := bufio.NewReader(os.Stdin)
				input, _ := reader.ReadString('\n')
				if strings.TrimSpace(input) != "Y" && strings.TrimSpace(input) != "y" {
					return nil
				}
			}
		}
	}

	// Git Source Control
	fmt.Println(ui.RenderSectionTitle("Git Source Control"))
	if target == "local" {
		ui.PrintInfo("Building from local source...")
	} else {
		targetBranch := target
		targetCommit := "latest"
		if strings.Contains(target, "@") {
			parts := strings.Split(target, "@")
			targetBranch = parts[0]
			targetCommit = parts[1]
		}

		ui.PrintInfo(fmt.Sprintf("Update target: %s @ %s", targetBranch, targetCommit))

		// Check if target branch exists
		if err := git.SwitchBranch(dexCliPath, targetBranch); err != nil {
			return err
		}
		ui.PrintInfo(fmt.Sprintf("Switched to branch '%s'", targetBranch))

		if targetCommit == "latest" {
			if err := git.Pull(dexCliPath); err != nil {
				return err
			}
			ui.PrintInfo("Pulled latest changes")
		} else {
			if err := git.CheckoutCommit(dexCliPath, targetCommit); err != nil {
				return err
			}
			ui.PrintInfo(fmt.Sprintf("Checked out commit '%s'", targetCommit))
		}
	}

	// Build & Install
	fmt.Println(ui.RenderSectionTitle("Build & Install"))
	ui.PrintInfo("Building `dex-cli` from source...")
	if err := buildDexCLI(dexCliPath); err != nil {
		return err
	}
	ui.PrintInfo("Installing `dex-cli` to `~/Dexter/bin`...")
	if err := installDexCLI(dexCliPath); err != nil {
		return err
	}
	ui.PrintInfo("Post-install processes...")
	newVersion, _ := exec.Command("dex", "version").Output()
	ui.PrintSuccess(fmt.Sprintf("Update complete. New version: %s", strings.TrimSpace(string(newVersion))))

	return nil
}

func buildDexCLI(dexCliPath string) error {
	cmd := exec.Command("go", "build", "-o", "dex-cli", ".")
	cmd.Dir = dexCliPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func installDexCLI(dexCliPath string) error {
	sourcePath := filepath.Join(dexCliPath, "dex-cli")
	destPath, err := config.ExpandPath("~/Dexter/bin/dex")
	if err != nil {
		return err
	}
	return os.Rename(sourcePath, destPath)
}
