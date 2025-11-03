package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/ui"
)

// Pull clones and updates services
func Pull(args []string) error {
	serviceMap, err := config.LoadServiceMapConfig()
	if err != nil {
		return fmt.Errorf("failed to load service map: %w", err)
	}

	for serviceType, services := range serviceMap.Services {
		if serviceType == "os" {
			continue
		}

		for _, service := range services {
			if service.ID == "dex-cli" {
				continue
			}

			if service.Source == "" || service.Repo == "" {
				continue
			}

			sourcePath, err := config.ExpandPath(service.Source)
			if err != nil {
				ui.PrintError(fmt.Sprintf("failed to expand source path for %s: %v", service.ID, err))
				continue
			}

			if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
				ui.PrintInfo(fmt.Sprintf("Source for %s not found, cloning from %s...", service.ID, service.Repo))
				cmd := exec.Command("git", "clone", "--depth", "1", "--branch", "main", service.Repo, sourcePath)
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				if err := cmd.Run(); err != nil {
					ui.PrintError(fmt.Sprintf("failed to clone %s: %v", service.ID, err))
				}
			} else {
				ui.PrintInfo(fmt.Sprintf("Updating %s...", service.ID))
				if err := forcePull(sourcePath); err != nil {
					ui.PrintError(fmt.Sprintf("failed to update %s: %v", service.ID, err))
				}
			}
		}
	}

	ui.PrintInfo("All services updated.")
	return nil
}

func forcePull(path string) error {
	cmd := exec.Command("git", "fetch", "--all")
	cmd.Dir = path
	if err := cmd.Run(); err != nil {
		return err
	}

	cmd = exec.Command("git", "reset", "--hard", "origin/main")
	cmd.Dir = path
	if err := cmd.Run(); err != nil {
		return err
	}

	cmd = exec.Command("git", "pull", "origin", "main")
	cmd.Dir = path
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}
