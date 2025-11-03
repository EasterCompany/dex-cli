package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Resolve finds a service's static definition by its shorthand alias.
// This is the primary function for validating and retrieving service info.
func Resolve(shortName string) (ServiceDefinition, error) {
	if def, ok := ServiceDefinitions[shortName]; ok {
		return def, nil
	}
	return ServiceDefinition{}, fmt.Errorf("service alias '%s' is not recognized", shortName)
}

// IsValidService checks if a shorthand alias exists.
func IsValidService(shortName string) bool {
	_, ok := ServiceDefinitions[shortName]
	return ok
}

// IsManageable checks if a service is controllable via systemd (start/stop/logs).
func IsManageable(shortName string) bool {
	if def, ok := ServiceDefinitions[shortName]; ok {
		// cli and os services are not managed by systemd
		return def.Type != "cli" && def.Type != "os"
	}
	return false
}

// GetSystemdName constructs the systemd service name from a definition.
// e.g., "dex-event-service" -> "dex-event.service"
func (def *ServiceDefinition) GetSystemdName() string {
	systemdName := strings.TrimSuffix(def.ID, "-service")
	return fmt.Sprintf("%s.service", systemdName)
}

// GetLogPath constructs the log file path from a definition.
func (def *ServiceDefinition) GetLogPath() (string, error) {
	logName := fmt.Sprintf("%s.log", def.ID)
	return ExpandPath(filepath.Join(DexterLogs, logName))
}

// CheckSystemdService checks if the systemd service file actually exists for the user.
func (def *ServiceDefinition) CheckSystemdService() (bool, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false, fmt.Errorf("failed to get home directory: %w", err)
	}
	systemdServicePath := filepath.Join(homeDir, ".config", "systemd", "user", def.GetSystemdName())

	_, err = os.Stat(systemdServicePath)
	if err == nil {
		return true, nil // File exists
	}
	if os.IsNotExist(err) {
		return false, nil // File does not exist
	}
	return false, err // Other error (e.g., permissions)
}

// GetManageableServices returns a sorted list of all manageable service aliases.
func GetManageableServices() []string {
	var aliases []string
	for alias, def := range ServiceDefinitions {
		if def.Type != "cli" && def.Type != "os" {
			aliases = append(aliases, alias)
		}
	}
	sort.Strings(aliases)
	return aliases
}

// GetBuildableServices returns a sorted list of all buildable service aliases.
func GetBuildableServices() []string {
	var aliases []string
	for alias, def := range ServiceDefinitions {
		// Only services with a defined Source path can be built
		if def.Source != "" && def.Type != "cli" {
			aliases = append(aliases, alias)
		}
	}
	sort.Strings(aliases)
	return aliases
}
