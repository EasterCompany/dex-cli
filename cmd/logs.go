package cmd

import (
	"fmt"
	"os"
	"os/exec"

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

	var servicesToShow []string
	if len(args) == 0 || (len(args) > 0 && args[0] == "all") {
		// Get all manageable service aliases
		servicesToShow = config.GetManageableServices()
	} else {
		// Validate each alias
		for _, arg := range args {
			if !config.IsManageable(arg) {
				return fmt.Errorf("service alias '%s' is not recognized or cannot be logged", arg)
			}
			servicesToShow = append(servicesToShow, arg)
		}
	}

	// Show logs for the selected services
	logFiles := []string{}
	for _, alias := range servicesToShow {
		def, _ := config.Resolve(alias) // Error already checked
		logPath, err := def.GetLogPath()
		if err != nil {
			return fmt.Errorf("failed to expand log path for '%s': %w", alias, err)
		}

		if _, err := os.Stat(logPath); os.IsNotExist(err) {
			log(fmt.Sprintf("Log file for service '%s' not found at %s, creating it.", alias, logPath))
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
	if len(logFiles) == 0 {
		return fmt.Errorf("no manageable services found to log")
	}

	tailArgs = append(tailArgs, logFiles...)
	cmd := exec.Command("tail", tailArgs...)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
