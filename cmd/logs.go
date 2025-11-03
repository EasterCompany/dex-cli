package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/ui"
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

	// Determine which services to show logs for
	servicesToShow := []config.ServiceDefinition{}
	if len(args) == 0 || (len(args) > 0 && args[0] == "all") {
		servicesToShow = config.GetManageableServices()
	} else {
		for _, arg := range args {
			serviceDef, err := config.Resolve(arg)
			if err != nil {
				return fmt.Errorf("failed to resolve service '%s': %w", arg, err)
			}
			if !serviceDef.IsManageable() {
				ui.PrintWarning(fmt.Sprintf("Skipping logs for non-manageable service: %s", serviceDef.ShortName))
				continue
			}
			servicesToShow = append(servicesToShow, *serviceDef)
		}
	}

	// Show logs for the selected services
	logFiles := []string{}
	for _, serviceDef := range servicesToShow {
		logPath, err := serviceDef.GetLogPath()
		if err != nil {
			return fmt.Errorf("failed to get log path for %s: %w", serviceDef.ShortName, err)
		}

		if _, err := os.Stat(logPath); os.IsNotExist(err) {
			log(fmt.Sprintf("Log file for service '%s' not found at %s, creating it.", serviceDef.ID, logPath))
			// Create the directory
			if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
				return fmt.Errorf("failed to create log directory: %w", err)
			}
			// Create the file
			if _, err := os.Create(logPath); err != nil {
				return fmt.Errorf("failed to create log file: %w", err)
			}
		}
		logFiles = append(logFiles, logPath)
	}

	if len(logFiles) == 0 {
		ui.PrintInfo("No services selected or no manageable services found.")
		return nil
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
