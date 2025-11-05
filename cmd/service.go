package cmd

import (
	"fmt"
	"os/exec"
	"sync"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/ui"
)

// Service handles start, stop, and restart commands for all manageable services.
func Service(command string) error {
	ui.PrintInfo(fmt.Sprintf("Attempting to %s all services...", command))

	serviceMap, err := config.LoadServiceMapConfig()
	if err != nil {
		return fmt.Errorf("failed to load service-map.json: %w", err)
	}

	var servicesToManage []config.ServiceDefinition
	for _, serviceList := range serviceMap.Services {
		for _, serviceEntry := range serviceList {
			def := config.GetServiceDefinition(serviceEntry.ID)
			// Only manage services that are not "cli" or "os" and have a systemd name
			if def.ID != "" && def.Type != "cli" && def.Type != "os" && def.SystemdName != "" {
				servicesToManage = append(servicesToManage, def)
			}
		}
	}

	if len(servicesToManage) == 0 {
		ui.PrintInfo("No manageable services found in service-map.json.")
		return nil
	}

	var wg sync.WaitGroup
	errors := make(chan error, len(servicesToManage))

	for _, s := range servicesToManage {
		wg.Add(1)
		go func(service config.ServiceDefinition) {
			defer wg.Done()
			ui.PrintInfo(fmt.Sprintf("Executing '%s' for %s...", command, service.ShortName))
			cmd := exec.Command("systemctl", "--user", command, service.SystemdName)
			if output, err := cmd.CombinedOutput(); err != nil {
				errors <- fmt.Errorf("failed to %s %s: %s", command, service.ShortName, string(output))
			}
		}(s)
	}

	wg.Wait()
	close(errors)

	hasErrors := false
	for err := range errors {
		hasErrors = true
		ui.PrintError(err.Error())
	}

	if hasErrors {
		return fmt.Errorf("one or more services failed to %s", command)
	}

	ui.PrintSuccess(fmt.Sprintf("Successfully executed '%s' for all services.", command))
	return nil
}
