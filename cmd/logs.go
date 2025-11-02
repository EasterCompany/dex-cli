package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/ui"
)

// Logs displays logs for a given service
func Logs(service string, follow bool) error {
	serviceMap, err := config.LoadServiceMap()
	if err != nil {
		return fmt.Errorf("failed to load service map: %w", err)
	}

	shorthandMap := make(map[string]string)
	for _, services := range serviceMap.Services {
		for _, s := range services {
			if len(s.ID) > 10 && s.ID[:4] == "dex-" && s.ID[len(s.ID)-8:] == "-service" {
				shorthand := s.ID[4 : len(s.ID)-8]
				shorthandMap[shorthand] = s.ID
			}
		}
	}

	fullServiceID := ""
	// Check for full service ID match first
	for _, services := range serviceMap.Services {
		for _, s := range services {
			if s.ID == service {
				fullServiceID = s.ID
				break
			}
		}
		if fullServiceID != "" {
			break
		}
	}

	// If no full ID match, check for shorthand match
	if fullServiceID == "" {
		if id, ok := shorthandMap[service]; ok {
			fullServiceID = id
		}
	}

	if fullServiceID == "" {
		availableServices := []string{}
		for _, services := range serviceMap.Services {
			for _, s := range services {
				availableServices = append(availableServices, s.ID)
			}
		}
		for shorthand := range shorthandMap {
			availableServices = append(availableServices, shorthand)
		}
		return fmt.Errorf("invalid service name: %s. Available services and shorthands: %v", service, availableServices)
	}

	logPath, err := config.ExpandPath(fmt.Sprintf("~/Dexter/logs/%s.log", fullServiceID))
	if err != nil {
		return fmt.Errorf("failed to expand log path: %w", err)
	}

	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		ui.PrintInfo(fmt.Sprintf("Log file for service '%s' not found at %s, creating it.", fullServiceID, logPath))
		if _, err := os.Create(logPath); err != nil {
			return fmt.Errorf("failed to create log file: %w", err)
		}
	}

	var cmd *exec.Cmd
	if follow {
		cmd = exec.Command("tail", "-f", logPath)
	} else {
		cmd = exec.Command("tail", logPath)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
