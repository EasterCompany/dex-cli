package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// ResolveService takes a short-hand service name and attempts to resolve it to
// a full systemd service name (e.g., "dex-discord.service") and a full
// project directory name (e.g., "dex-discord-service").
// It also handles the special case of "cli" for the dex-cli project.
func ResolveService(shortName string) (systemdName string, projectDirName string, err error) {
	if shortName == "cli" {
		// Special case for dex-cli, which is not a systemd service but a project
		return "", "dex-cli", nil
	}

	// Try to resolve as a systemd service
	potentialSystemdName := fmt.Sprintf("dex-%s.service", shortName)
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", "", fmt.Errorf("failed to get home directory: %w", err)
	}
	systemdServicePath := filepath.Join(homeDir, ".config", "systemd", "user", potentialSystemdName)

	_, err = os.Stat(systemdServicePath)
	if err == nil {
		// Found a systemd service
		return potentialSystemdName, fmt.Sprintf("dex-%s-service", shortName), nil
	}

	// If not a systemd service, try to resolve as a project directory
	potentialProjectDirName := fmt.Sprintf("dex-%s-service", shortName)
	easterCompanyPath, err := ExpandPath(EasterCompanyRoot)
	if err != nil {
		return "", "", fmt.Errorf("failed to expand EasterCompany root path: %w", err)
	}
	projectDirPath := filepath.Join(easterCompanyPath, potentialProjectDirName)

	_, err = os.Stat(projectDirPath)
	if err == nil {
		// Found a project directory, derive systemd name from it
		derivedSystemdName := fmt.Sprintf("dex-%s.service", shortName)
		return derivedSystemdName, potentialProjectDirName, nil
	}

	return "", "", fmt.Errorf("service '%s' not found as a systemd service or project directory", shortName)
}

// ResolveSystemdService takes a short-hand service name and resolves it to a full systemd service name.
// It returns an error if the service is not found or is not a systemd service.
func ResolveSystemdService(shortName string) (string, error) {
	systemdName, _, err := ResolveService(shortName)
	if err != nil {
		return "", err
	}
	if systemdName == "" {
		return "", fmt.Errorf("service '%s' is not a systemd service", shortName)
	}
	return systemdName, nil
}

// ResolveProjectDirService takes a short-hand service name and resolves it to a full project directory name.
// It returns an error if the service is not found or is not a project directory.
func ResolveProjectDirService(shortName string) (string, error) {
	_, projectDirName, err := ResolveService(shortName)
	if err != nil {
		return "", err
	}
	if projectDirName == "" {
		return "", fmt.Errorf("service '%s' is not a project directory", shortName)
	}
	return projectDirName, nil
}
