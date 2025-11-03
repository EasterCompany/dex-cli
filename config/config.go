package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Hardcoded Dexter environment layout
const (
	DexterRoot        = "~/Dexter"
	EasterCompanyRoot = "~/EasterCompany"
)

// Required subdirectories in ~/Dexter
var RequiredDexterDirs = []string{
	"config",
	"models",
	"bin",
	"run",
	"logs",
}

// ExpandPath expands ~ to the user's home directory
func ExpandPath(path string) (string, error) {
	if !strings.HasPrefix(path, "~") {
		return path, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	if path == "~" {
		return homeDir, nil
	}

	return filepath.Join(homeDir, path[2:]), nil
}

// EnsureDirectoryStructure creates required directories if they don't exist
func EnsureDirectoryStructure() error {
	// Ensure ~/Dexter exists
	dexterPath, err := ExpandPath(DexterRoot)
	if err != nil {
		return fmt.Errorf("failed to expand Dexter root path: %w", err)
	}

	if err := os.MkdirAll(dexterPath, 0755); err != nil {
		return fmt.Errorf("failed to create Dexter directory: %w", err)
	}

	// Ensure all required subdirectories exist
	for _, dir := range RequiredDexterDirs {
		dirPath := filepath.Join(dexterPath, dir)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return fmt.Errorf("failed to create %s directory: %w", dir, err)
		}
	}

	// Ensure ~/EasterCompany exists
	easterCompanyPath, err := ExpandPath(EasterCompanyRoot)
	if err != nil {
		return fmt.Errorf("failed to expand EasterCompany root path: %w", err)
	}

	if err := os.MkdirAll(easterCompanyPath, 0755); err != nil {
		return fmt.Errorf("failed to create EasterCompany directory: %w", err)
	}

	return nil
}

// EnsureConfigFiles creates and validates all config files.
func EnsureConfigFiles() error {
	// Service Map
	_, err := LoadServiceMapConfig()
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("Creating default service-map.json...")
			if err := SaveServiceMapConfig(DefaultServiceMapConfig()); err != nil {
				return fmt.Errorf("failed to save default service-map.json: %w", err)
			}
		} else {
			return fmt.Errorf("failed to load service-map.json: %w", err)
		}
	}

	// Options
	_, err = LoadOptionsConfig()
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("Creating default options.json...")
			if err := SaveOptionsConfig(DefaultOptionsConfig()); err != nil {
				return fmt.Errorf("failed to save default options.json: %w", err)
			}
		} else {
			return fmt.Errorf("failed to load options.json: %w", err)
		}
	}

	return nil
}

// LogFile returns a file handle to the dex-cli log file.
func LogFile() (*os.File, error) {
	logPath, err := ExpandPath(filepath.Join(DexterRoot, "logs", "dex-cli.log"))
	if err != nil {
		return nil, fmt.Errorf("failed to expand log file path: %w", err)
	}

	// Ensure the directory exists.
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open the file in append mode, create it if it doesn't exist.
	file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	return file, nil
}

// HasSourceServices checks if the EasterCompany source directory exists.
func IsDevMode() bool {
	// Check if the source code directory exists
	path, err := ExpandPath("~/EasterCompany/dex-cli")
	if err != nil {
		return false
	}
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}
