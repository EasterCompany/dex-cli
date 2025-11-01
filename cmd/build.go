package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/ui"
)

// Build compiles one or all services
func Build(serviceName string) error {
	ui.PrintTitle("DEXTER BUILD COMMAND")

	// Load the service map
	ui.PrintSectionTitle("LOADING SERVICE MAP")
	serviceMap, err := config.LoadServiceMap()
	if err != nil {
		return fmt.Errorf("failed to load service map: %w", err)
	}
	ui.PrintSuccess("Service map loaded")

	// Build logic
	if serviceName == "all" {
		return buildAll(serviceMap)
	} else if serviceName == "self" {
		return buildSelf()
	} else {
		return buildOne(serviceMap, serviceName)
	}
}

func buildAll(serviceMap *config.ServiceMapConfig) error {
	ui.PrintSectionTitle("BUILDING ALL SERVICES")
	for _, services := range serviceMap.Services {
		for _, service := range services {
			if err := buildService(service); err != nil {
				ui.PrintError(fmt.Sprintf("Failed to build %s: %v", service.ID, err))
			}
		}
	}
	ui.PrintSuccess("All services built")
	return nil
}

func buildOne(serviceMap *config.ServiceMapConfig, serviceName string) error {
	ui.PrintSectionTitle(fmt.Sprintf("BUILDING SERVICE: %s", serviceName))
	for _, services := range serviceMap.Services {
		for _, service := range services {
			if service.ID == serviceName {
				return buildService(service)
			}
		}
	}
	return fmt.Errorf("service '%s' not found in service-map.json", serviceName)
}

func buildSelf() error {
	ui.PrintSectionTitle("BUILDING DEX-CLI")

	// Get git commit hash
	commitCmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	commitCmd.Dir, _ = config.ExpandPath("~/EasterCompany/dex-cli")
	commit, err := commitCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get git commit: %w", err)
	}

	// Get current date
	date := time.Now().UTC().Format("2006-01-02T15:04:05Z")

	dexterBinPath, err := config.ExpandPath("~/Dexter/bin")
	if err != nil {
		return err
	}

	ldflags := fmt.Sprintf("-X main.commit=%s -X main.date=%s", strings.TrimSpace(string(commit)), date)

	cmd := exec.Command("go", "build", "-ldflags", ldflags, "-o", filepath.Join(dexterBinPath, "dex"))
	cmd.Dir, _ = config.ExpandPath("~/EasterCompany/dex-cli")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to build dex-cli: %w", err)
	}
	ui.PrintSuccess("dex-cli built successfully")
	return nil
}

func buildService(service config.ServiceEntry) error {
	if service.Source == "" || service.Source == "system" {
		ui.PrintWarning(fmt.Sprintf("Skipping %s: no source path defined", service.ID))
		return nil
	}

	sourcePath, err := config.ExpandPath(service.Source)
	if err != nil {
		return fmt.Errorf("failed to expand source path for %s: %w", service.ID, err)
	}

	if _, err := os.Stat(filepath.Join(sourcePath, "go.mod")); os.IsNotExist(err) {
		ui.PrintWarning(fmt.Sprintf("Skipping %s: not a Go project (no go.mod)", service.ID))
		return nil
	}

	ui.PrintInfo(fmt.Sprintf("Building %s...", service.ID))

	dexterBinPath, err := config.ExpandPath("~/Dexter/bin")
	if err != nil {
		return err
	}

	cmd := exec.Command("go", "build", "-o", filepath.Join(dexterBinPath, service.ID))
	cmd.Dir = sourcePath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to build %s: %w", service.ID, err)
	}

	ui.PrintSuccess(fmt.Sprintf("%s built successfully", service.ID))
	return nil
}
