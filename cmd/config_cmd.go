package cmd

import (
	"fmt"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/ui"
)

// Config is the entry point for the config command
func Config(subcommand string) error {
	switch subcommand {
	case "validate":
		return ValidateConfig()
	default:
		return fmt.Errorf("unknown config subcommand: %s", subcommand)
	}
}

// ValidateConfig checks all configuration files for correctness
func ValidateConfig() error {
	ui.PrintTitle("DEXTER CONFIG VALIDATE")

	// Validate service-map.json
	ui.PrintSectionTitle("VALIDATING service-map.json")
	if err := validateServiceMap(); err != nil {
		ui.PrintError(fmt.Sprintf("service-map.json validation failed: %v", err))
	} else {
		ui.PrintSuccess("service-map.json is valid")
	}

	// Validate system.json
	ui.PrintSectionTitle("VALIDATING system.json")
	if err := validateSystemConfig(); err != nil {
		ui.PrintError(fmt.Sprintf("system.json validation failed: %v", err))
	} else {
		ui.PrintSuccess("system.json is valid")
	}

	// Validate options.json
	ui.PrintSectionTitle("VALIDATING options.json")
	if err := validateOptions(); err != nil {
		ui.PrintError(fmt.Sprintf("options.json validation failed: %v", err))
	} else {
		ui.PrintSuccess("options.json is valid")
	}

	return nil
}

func validateServiceMap() error {
	serviceMap, err := config.LoadServiceMap()
	if err != nil {
		return err
	}

	for serviceType, services := range serviceMap.Services {
		for _, service := range services {
			if service.ID == "" {
				ui.PrintWarning(fmt.Sprintf("Service in type '%s' is missing an ID", serviceType))
			}
			if service.Source == "" && serviceType != "os" {
				ui.PrintWarning(fmt.Sprintf("Service '%s' is missing a source", service.ID))
			}
			if service.Repo == "" && serviceType != "os" {
				ui.PrintWarning(fmt.Sprintf("Service '%s' is missing a repo", service.ID))
			}
		}
	}

	return nil
}

func validateSystemConfig() error {
	sys, err := config.LoadSystemConfig()
	if err != nil {
		return err
	}

	for _, pkg := range sys.Packages {
		if pkg.Required && !pkg.Installed {
			ui.PrintWarning(fmt.Sprintf("Required package '%s' is not installed", pkg.Name))
		}
	}

	return nil
}

func validateOptions() error {
	// This is a placeholder for options.json validation
	// In a real implementation, we would load options.json and check its fields
	ui.PrintInfo("options.json validation is not fully implemented yet.")
	return nil
}
