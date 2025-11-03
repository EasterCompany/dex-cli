package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/EasterCompany/dex-cli/config"
)

// Logs displays logs for a given service
func Logs(args []string, follow bool) error {
	logFile, err := config.LogFile()
	if err != nil {
		return fmt.Errorf("failed to get log file: %w", err)
	}
	defer func() { _ = logFile.Close() }()

	log := func(message string) {
		_, _ = fmt.Fprintln(logFile, message)
	}

	log(fmt.Sprintf("Displaying logs for services: %v, follow: %t", args, follow))

	serviceMap, err := config.LoadServiceMapConfig()
	if err != nil {
		return fmt.Errorf("failed to load service map: %w", err)
	}

	// Determine which services to show logs for
	servicesToShow := []string{}
	if len(args) == 0 || (len(args) > 0 && args[0] == "all") {
		for _, services := range serviceMap.Services {
			for _, service := range services {
				if strings.HasPrefix(service.ID, "dex-") {
					servicesToShow = append(servicesToShow, service.ID)
				}
			}
		}
	} else {
		for _, arg := range args {
			serviceName := arg
			if !strings.HasPrefix(serviceName, "dex-") {
				serviceName = "dex-" + serviceName + "-service"
			}
			found := false
			for _, services := range serviceMap.Services {
				for _, service := range services {
					if service.ID == serviceName {
						servicesToShow = append(servicesToShow, service.ID)
						found = true
						break
					}
				}
				if found {
					break
				}
			}
			if !found {
				return fmt.Errorf("service '%s' not found", arg)
			}
		}
	}

	// Show logs for the selected services
	logFiles := []string{}
	for _, serviceID := range servicesToShow {
		logPath, err := config.ExpandPath(fmt.Sprintf("~/Dexter/logs/%s.log", serviceID))
		if err != nil {
			return fmt.Errorf("failed to expand log path: %w", err)
		}

		if _, err := os.Stat(logPath); os.IsNotExist(err) {
			log(fmt.Sprintf("Log file for service '%s' not found at %s, creating it.", serviceID, logPath))
			if _, err := os.Create(logPath); err != nil {
				return fmt.Errorf("failed to create log file: %w", err)
			}
		}
		logFiles = append(logFiles, logPath)
	}

	tailArgs := []string{}
	if follow {
		tailArgs = append(tailArgs, "-f")
	}
	tailArgs = append(tailArgs, logFiles...)
	cmd := exec.Command("tail", tailArgs...)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
