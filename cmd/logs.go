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

	// Determine which services to show logs for
	servicesToShow := []config.ServiceDefinition{}
	if len(args) == 0 || (len(args) > 0 && args[0] == "all") {
		servicesToShow = config.GetManageableServices()
	} else {
		for _, arg := range args {
			def, err := config.Resolve(arg)
			if err != nil {
				return fmt.Errorf("failed to resolve service '%s': %w", arg, err)
			}
			if !def.IsManageable() {
				return fmt.Errorf("cannot show logs for non-manageable service '%s'", arg)
			}
			servicesToShow = append(servicesToShow, *def)
		}
	}

	// Show logs for the selected services
	logFiles := []string{}
	for _, serviceDef := range servicesToShow {
		logPathStr := serviceDef.GetLogPath()
		logPath, err := config.ExpandPath(logPathStr)
		if err != nil {
			return fmt.Errorf("failed to expand log path: %w", err)
		}

		if _, err := os.Stat(logPath); os.IsNotExist(err) {
			log(fmt.Sprintf("Log file for service '%s' not found at %s, creating it.", serviceDef.ShortName, logPath))
			if _, err := os.Create(logPath); err != nil {
				return fmt.Errorf("failed to create log file: %w", err)
			}
		}
		logFiles = append(logFiles, logPath)
	}

	if len(logFiles) == 0 {
		fmt.Println("No logs found for specified services.")
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
