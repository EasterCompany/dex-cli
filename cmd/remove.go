package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/ui"
)

// checkFileExists is a helper to check if a file or directory exists.
func checkFileExists(path string) bool {
	if path == "" || strings.HasPrefix(path, "~") && (path == "~" || path == "~/") {
		return false
	}
	expandedPath, err := config.ExpandPath(path)
	if err != nil {
		return false
	}
	_, err = os.Stat(expandedPath)
	return err == nil
}

// isServiceInMap is a helper to check if a service exists in the service-map.json.
func isServiceInMap(def config.ServiceDefinition, serviceMap *config.ServiceMapConfig) bool {
	for _, serviceType := range serviceMap.Services {
		for _, service := range serviceType {
			if service.ID == def.ID {
				return true
			}
		}
	}
	return false
}

// Remove a service from the service map and uninstall it from systemd
func Remove(args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("remove command takes no arguments")
	}

	serviceMap, err := config.LoadServiceMapConfig()
	if err != nil {
		return fmt.Errorf("failed to load service map: %w", err)
	}

	var removableServices []config.ServiceDefinition
	allServices := config.GetAllServices()
	sort.Slice(allServices, func(i, j int) bool {
		return allServices[i].ShortName < allServices[j].ShortName
	})

	// Track-and-trace: Find which services have any artifacts on the system
	for _, def := range allServices {
		if !def.IsManageable() {
			continue
		}

		// Check all possible artifact locations
		hasArtifacts := false
		if checkFileExists(def.GetSystemdPath()) {
			hasArtifacts = true
		}
		if !hasArtifacts && checkFileExists(def.GetLogPath()) {
			hasArtifacts = true
		}
		if !hasArtifacts && checkFileExists(def.Source) {
			hasArtifacts = true
		}
		if !hasArtifacts && checkFileExists(def.GetBinaryPath()) {
			hasArtifacts = true
		}
		if !hasArtifacts && isServiceInMap(def, serviceMap) {
			hasArtifacts = true
		}

		if hasArtifacts {
			removableServices = append(removableServices, def)
		}
	}

	if len(removableServices) == 0 {
		ui.PrintInfo("No removable services found on the system.")
		return nil
	}

	fmt.Println("Available services to remove:")
	for i, def := range removableServices {
		fmt.Printf("  %d: %s (ID: %s, Type: %s)\n", i+1, def.ShortName, def.ID, def.Type)
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter number(s) of services to remove (e.g., '1' or '1,2'): ")
	input, _ := reader.ReadString('\n')

	selectedIndices, err := parseNumericInput(input, len(removableServices))
	if err != nil {
		return fmt.Errorf("invalid input: %w", err)
	}

	if len(selectedIndices) == 0 {
		ui.PrintInfo("No services selected.")
		return nil
	}

	for _, index := range selectedIndices {
		def := removableServices[index]
		fmt.Println(ui.Colorize(fmt.Sprintf("--- Removing %s ---", def.ShortName), ui.ColorCyan))

		// 1. Stop the service
		if err := stopSystemdService(def.SystemdName); err != nil {
			ui.PrintWarning(fmt.Sprintf("Could not stop service (may already be stopped): %v", err))
		} else {
			ui.PrintInfo(fmt.Sprintf("Stopped service: %s", def.SystemdName))
		}

		// 2. Disable the service
		if err := disableSystemdService(def.SystemdName); err != nil {
			ui.PrintWarning(fmt.Sprintf("Could not disable service: %v", err))
		} else {
			ui.PrintInfo(fmt.Sprintf("Disabled service: %s", def.SystemdName))
		}

		// 3. Remove the .service file
		if err := removeSystemdFile(def.GetSystemdPath()); err != nil {
			ui.PrintWarning(fmt.Sprintf("Could not remove systemd file: %v", err))
		} else {
			ui.PrintInfo(fmt.Sprintf("Removed systemd file: %s", def.GetSystemdPath()))
		}

		// 4. Remove the log file
		if err := removeLogFile(def.GetLogPath()); err != nil {
			ui.PrintWarning(fmt.Sprintf("Could not remove log file: %v", err))
		} else {
			ui.PrintInfo(fmt.Sprintf("Removed log file: %s", def.GetLogPath()))
		}

		// 5. Remove Source Directory
		if err := removeSourceDirectory(def.Source); err != nil {
			ui.PrintWarning(fmt.Sprintf("Could not remove source directory: %v", err))
		} else {
			ui.PrintInfo(fmt.Sprintf("Removed source directory: %s", def.Source))
		}

		// 6. Remove Binary File
		if err := removeBinaryFile(def.GetBinaryPath()); err != nil {
			ui.PrintWarning(fmt.Sprintf("Could not remove binary file: %v", err))
		} else {
			ui.PrintInfo(fmt.Sprintf("Removed binary file: %s", def.GetBinaryPath()))
		}

		// 7. Remove from service-map.json
		found := false
		for serviceType, services := range serviceMap.Services {
			for i, service := range services {
				if service.ID == def.ID {
					serviceMap.Services[serviceType] = append(services[:i], services[i+1:]...)
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			ui.PrintWarning(fmt.Sprintf("Service '%s' not found in service-map.json (was already removed?).", def.ID))
		} else {
			ui.PrintInfo(fmt.Sprintf("Removed '%s' from service-map.json", def.ShortName))
		}
	}

	// 8. Reload systemd daemon
	if err := reloadSystemdDaemon(); err != nil {
		ui.PrintWarning(fmt.Sprintf("Failed to reload systemd daemon: %v", err))
	} else {
		ui.PrintInfo("Reloaded systemd daemon")
	}

	// 9. Save the updated service map
	if err := config.SaveServiceMapConfig(serviceMap); err != nil {
		return fmt.Errorf("failed to save service map: %w", err)
	}

	ui.PrintSuccess(fmt.Sprintf("Successfully removed %d service(s).", len(selectedIndices)))
	return nil
}

func parseNumericInput(input string, maxIndex int) ([]int, error) {
	input = strings.TrimSpace(input)
	input = strings.TrimSuffix(input, ",")
	if input == "" {
		return nil, nil
	}

	parts := strings.Split(input, ",")
	var indices []int
	seen := make(map[int]bool)

	for _, part := range parts {
		numStr := strings.TrimSpace(part)
		if numStr == "" {
			continue
		}

		num, err := strconv.Atoi(numStr)
		if err != nil {
			return nil, fmt.Errorf("'%s' is not a valid number", numStr)
		}

		index := num - 1 // Convert 1-based to 0-based
		if index < 0 || index >= maxIndex {
			return nil, fmt.Errorf("number %d is out of range (must be 1-%d)", num, maxIndex)
		}

		if !seen[index] {
			indices = append(indices, index)
			seen[index] = true
		}
	}
	return indices, nil
}

func stopSystemdService(systemdName string) error {
	cmd := exec.Command("systemctl", "--user", "stop", systemdName)
	return cmd.Run()
}

func disableSystemdService(systemdName string) error {
	cmd := exec.Command("systemctl", "--user", "disable", systemdName)
	return cmd.Run()
}

func removeSystemdFile(systemdPath string) error {
	if _, err := os.Stat(systemdPath); os.IsNotExist(err) {
		return nil // File already gone
	}
	return os.Remove(systemdPath)
}

func removeLogFile(logPath string) error {
	expandedPath, err := config.ExpandPath(logPath)
	if err != nil {
		return err
	}
	if _, err := os.Stat(expandedPath); os.IsNotExist(err) {
		return nil // File already gone
	}
	return os.Remove(expandedPath)
}

func removeSourceDirectory(sourcePath string) error {
	expandedPath, err := config.ExpandPath(sourcePath)
	if err != nil {
		return err
	}
	if _, err := os.Stat(expandedPath); os.IsNotExist(err) {
		return nil // Dir already gone
	}
	return os.RemoveAll(expandedPath)
}

func removeBinaryFile(binaryPath string) error {
	// GetBinaryPath() already returns an expanded path
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		return nil // File already gone
	}
	return os.Remove(binaryPath)
}

func reloadSystemdDaemon() error {
	cmd := exec.Command("systemctl", "--user", "daemon-reload")
	return cmd.Run()
}
