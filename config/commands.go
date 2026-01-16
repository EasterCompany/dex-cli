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
			Description: "Update dex-cli and all services",
			Check:       func() bool { return true }, // Always available
		},
		"build": {
			Name:        "build",
			Description: "Build and install all services from local source",
			Check:       HasDexCliSource,
		},
		"start": {
			Name:        "start",
			Description: "Start a service",
			Check:       HasAnySystemdService,
		},
		"stop": {
			Name:        "stop",
			Description: "Stop a service",
			Check:       HasAnySystemdService,
		},
		"restart": {
			Name:        "restart",
			Description: "Restart a service",
			Check:       HasAnySystemdService,
		},
		"status": {
			Name:        "status",
			Description: "Check the health of one or all services",
			Check:       func() bool { return true }, // Always available
		},
		"python": {
			Name:        "python",
			Description: "Manage Dexter's Python environment",
			Check:       HasPythonVenv,
		},
		"bun": {
			Name:        "bun",
			Description: "Proxy for the system bun executable",
			Check:       HasBun,
		},
		"bunx": {
			Name:        "bunx",
			Description: "Proxy for the system bunx executable",
			Check:       HasBun,
		},
		"ollama": {
			Name:        "ollama",
			Description: "Proxy for the system ollama executable",
			Check:       HasOllama,
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
		"agent": {
			Name:        "agent",
			Description: "Manage and trigger Dexter Agents",
			Check:       func() bool { return true }, // Always available
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
		"add": {
			Name:        "add",
			Description: "Add (clone, build, install) a new service",
			Check:       func() bool { return true }, // Always available
		},
		"remove": {
			Name:        "remove",
			Description: "Uninstall and remove a service",
			Check:       func() bool { return true }, // Always available
		},
		"cache": {
			Name:        "cache",
			Description: "Manage the local cache (local-cache-0)",
			Check:       func() bool { return true }, // Always available
		},
		"event": {
			Name:        "event",
			Description: "Interact with the event service",
			Check:       HasEventService,
		},
		"discord": {
			Name:        "discord",
			Description: "Interact with the discord service",
			Check:       HasDiscordService,
		},
		"config": {
			Name:        "config",
			Description: "Show the service-map.json config for a service",
			Check:       func() bool { return true }, // Always available
		},
		"whisper": {
			Name:        "whisper",
			Description: "Speech-to-text transcription with Whisper",
			Check:       HasPythonVenv,
		},
		"verify": {
			Name:        "verify",
			Description: "Run deep system diagnostics",
			Check:       func() bool { return true },
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

// HasSourceDirectory checks if ~/EasterCompany/ exists
func HasSourceDirectory() bool {
	path, err := ExpandPath("~/EasterCompany/")
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
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
		if entry.IsDir() && strings.HasPrefix(entry.Name(), "dex-cli") {
			return true
		}
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

// HasPythonVenv checks if the Python virtual environment is installed and set up at ~/Dexter/python3.14
func HasPythonVenv() bool {
	pythonVenvPath, err := ExpandPath(filepath.Join(DexterRoot, "python3.14"))
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

// HasOllama checks if 'ollama' executable is available in the system's PATH
func HasOllama() bool {
	_, err := exec.LookPath("ollama")
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

// HasEventService checks if dex-event-service is in service-map.json
func HasEventService() bool {
	serviceMap, err := LoadServiceMapConfig()
	if err != nil {
		return false // If we can't load the map, don't show the command
	}

	coreServices, ok := serviceMap.Services["cs"]
	if !ok {
		return false
	}

	for _, service := range coreServices {
		if service.ID == "dex-event-service" {
			return true
		}
	}
	return false
}

// HasDiscordService checks if dex-discord-service is in service-map.json
func HasDiscordService() bool {
	serviceMap, err := LoadServiceMapConfig()
	if err != nil {
		return false // If we can't load the map, don't show the command
	}

	thirdPartyServices, ok := serviceMap.Services["th"]
	if !ok {
		return false
	}

	for _, service := range thirdPartyServices {
		if service.ID == "dex-discord-service" {
			return true
		}
	}
	return false
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

// HasSourceServices checks if any dex services with source code exist.
func HasSourceServices() bool {
	easterCompanyPath, err := ExpandPath(EasterCompanyRoot)
	if err != nil {
		return false
	}

	if _, err := os.Stat(easterCompanyPath); os.IsNotExist(err) {
		return false
	}

	// Check for any dex-* directories (including dex-cli, dex-*-service, etc.)
	matches, err := filepath.Glob(filepath.Join(easterCompanyPath, "dex-*"))
	if err != nil {
		return false
	}

	// Filter to only directories
	for _, match := range matches {
		if info, err := os.Stat(match); err == nil && info.IsDir() {
			return true
		}
	}

	return false
}
