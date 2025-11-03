package config

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// CommandRequirement defines the requirements for a command to be available
type CommandRequirement struct {
	Name        string
	Description string
	Check       func() bool
}

// GetCommandRequirements returns all command requirements
func GetCommandRequirements() map[string]CommandRequirement {
	return map[string]CommandRequirement{
		"update": {
			Name:        "update",
			Description: "Update dex-cli to latest version",
			Check:       HasDexCliSource,
		},
		"build": {
			Name:        "build",
			Description: "Build one or all Dexter services",
			Check:       HasAnyDexService,
		},
		"start": {
			Name:        "start",
			Description: "Start a Dexter service",
			Check:       HasAnySystemdService,
		},
		"stop": {
			Name:        "stop",
			Description: "Stop a Dexter service",
			Check:       HasAnySystemdService,
		},
		"restart": {
			Name:        "restart",
			Description: "Restart a Dexter service",
			Check:       HasAnySystemdService,
		},
		"status": {
			Name:        "status",
			Description: "Check the health of one or all services",
			Check:       HasAnySystemdService,
		},
		"python": {
			Name:        "python",
			Description: "Manage Dexter's Python environment or run Python commands",
			Check:       HasPythonVenv,
		},
		"bun": {
			Name:        "bun",
			Description: "Proxy for the system's bun executable",
			Check:       HasBun,
		},
		"bunx": {
			Name:        "bunx",
			Description: "Proxy for the system's bunx executable",
			Check:       HasBun,
		},
		"logs": {
			Name:        "logs",
			Description: "View service logs",
			Check:       HasAnySystemdService,
		},
		"test": {
			Name:        "test",
			Description: "Run all tests",
			Check:       HasSourceServices,
		},
		"system": {
			Name:        "system",
			Description: "Show system info and manage packages",
			Check:       func() bool { return true }, // Always available
		},
		"version": {
			Name:        "version",
			Description: "Show version information",
			Check:       func() bool { return true }, // Always available
		},
		"help": {
			Name:        "help",
			Description: "Show this help message",
			Check:       func() bool { return true }, // Always available
		},
		"pull": {
			Name:        "pull",
			Description: "Pull latest changes for all Dexter services",
			Check:       HasEasterCompanyRoot,
		},
	}
}

// IsCommandAvailable checks if a command meets its requirements
func IsCommandAvailable(command string) bool {
	requirements := GetCommandRequirements()
	req, exists := requirements[command]
	if !exists {
		// Command has no requirements, always available
		return true
	}
	return req.Check()
}

// HasDexCliSource checks if ~/EasterCompany/dex-cli exists
func HasDexCliSource() bool {
	path, err := ExpandPath("~/EasterCompany/dex-cli")
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

// HasAnyDexService checks if any dex-*-service project exists in ~/EasterCompany
func HasAnyDexService() bool {
	easterPath, err := ExpandPath(EasterCompanyRoot)
	if err != nil {
		return false
	}

	entries, err := os.ReadDir(easterPath)
	if err != nil {
		return false
	}

	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), "dex-") && strings.HasSuffix(entry.Name(), "-service") {
			return true
		}
	}
	return false
}

// HasAnySystemdService checks if any dex-*.service exists in user's systemd directory
func HasAnySystemdService() bool {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false
	}

	systemdPath := filepath.Join(homeDir, ".config", "systemd", "user")
	entries, err := os.ReadDir(systemdPath)
	if err != nil {
		return false
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasPrefix(entry.Name(), "dex-") && strings.HasSuffix(entry.Name(), ".service") {
			return true
		}
	}
	return false
}

// HasPythonVenv checks if the Python virtual environment is installed and set up at ~/Dexter/python
func HasPythonVenv() bool {
	pythonVenvPath, err := ExpandPath(filepath.Join(DexterRoot, "python"))
	if err != nil {
		return false
	}
	// Check for the existence of the 'bin' directory inside the venv, which usually contains the python executable
	_, err = os.Stat(filepath.Join(pythonVenvPath, "bin"))
	return err == nil
}

// HasBun checks if 'bun' executable is available in the system's PATH
func HasBun() bool {
	_, err := exec.LookPath("bun")
	return err == nil
}

// HasEasterCompanyRoot checks if ~/EasterCompany exists
func HasEasterCompanyRoot() bool {
	path, err := ExpandPath(EasterCompanyRoot)
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

// GetDexServices returns a list of all dex-*-service directories in ~/EasterCompany
func GetDexServices() []string {
	easterPath, err := ExpandPath(EasterCompanyRoot)
	if err != nil {
		return nil
	}

	entries, err := os.ReadDir(easterPath)
	if err != nil {
		return nil
	}

	var services []string
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), "dex-") && strings.HasSuffix(entry.Name(), "-service") {
			services = append(services, entry.Name())
		}
	}
	return services
}

// GetSystemdServices returns a list of all dex-*.service files in user's systemd directory
func GetSystemdServices() []string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	systemdPath := filepath.Join(homeDir, ".config", "systemd", "user")
	entries, err := os.ReadDir(systemdPath)
	if err != nil {
		return nil
	}

	var services []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasPrefix(entry.Name(), "dex-") && strings.HasSuffix(entry.Name(), ".service") {
			// Remove .service suffix to get service name
			serviceName := strings.TrimSuffix(entry.Name(), ".service")
			services = append(services, serviceName)
		}
	}
	return services
}

// ServiceNameToSystemdName converts dex-discord-service to dex-discord
func ServiceNameToSystemdName(serviceName string) string {
	return strings.TrimSuffix(serviceName, "-service")
}

// SystemdNameToServiceName converts dex-discord to dex-discord-service
func SystemdNameToServiceName(systemdName string) string {
	if strings.HasSuffix(systemdName, "-service") {
		return systemdName
	}
	return systemdName + "-service"
}
