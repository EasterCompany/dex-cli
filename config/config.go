package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/EasterCompany/dex-cli/ui"
)

const (
	DexterRoot        = "~/Dexter"
	DexterBin         = "~/Dexter/bin"
	DexterLogs        = "~/Dexter/logs"
	EasterCompanyRoot = "~/EasterCompany"
)

var RequiredDexterDirs = []string{
	"bin",
	"config",
	"data",
	"logs",
	"models",
	"run",
}

// GetDexterPath returns the absolute path to the ~/Dexter directory.
func GetDexterPath() (string, error) {
	return ExpandPath(DexterRoot)
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

	if err := os.MkdirAll(dexterPath, 0o755); err != nil {
		return fmt.Errorf("failed to create Dexter directory: %w", err)
	}

	// Ensure all required subdirectories exist
	for _, dir := range RequiredDexterDirs {
		dirPath := filepath.Join(dexterPath, dir)
		if err := os.MkdirAll(dirPath, 0o755); err != nil {
			return fmt.Errorf("failed to create %s directory: %w", dir, err)
		}
	}

	// Ensure ~/EasterCompany exists
	easterCompanyPath, err := ExpandPath(EasterCompanyRoot)
	if err != nil {
		return fmt.Errorf("failed to expand EasterCompany root path: %w", err)
	}

	if err := os.MkdirAll(easterCompanyPath, 0o755); err != nil {
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

	// Options - with healing
	userOpts, err := LoadOptionsConfig()
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, create a new one with all defaults
			fmt.Println("Creating default options.json...")
			if err := SaveOptionsConfig(DefaultOptionsConfig()); err != nil {
				return fmt.Errorf("failed to save default options.json: %w", err)
			}
		} else {
			// Other error loading the file
			return fmt.Errorf("failed to load options.json: %w", err)
		}
	} else {
		// File exists, check if it needs healing
		if healed := healOptionsConfig(userOpts, DefaultOptionsConfig()); healed {
			fmt.Println("Healing options.json: Added missing default values.")
			if err := SaveOptionsConfig(userOpts); err != nil {
				return fmt.Errorf("failed to save healed options.json: %w", err)
			}
		}
	}

	// Server Map
	_, serverMapErr := LoadServerMapConfig()
	if serverMapErr != nil {
		if os.IsNotExist(serverMapErr) {
			fmt.Println("Creating default server-map.json...")
			if err := SaveServerMapConfig(DefaultServerMapConfig()); err != nil {
				return fmt.Errorf("failed to save default server-map.json: %w", err)
			}
		} else {
			return fmt.Errorf("failed to load server-map.json: %w", serverMapErr)
		}
	}

	return nil
}

// healOptionsConfig merges the default config into the user's config to add missing fields.
// It modifies the userOpts object directly. Returns true if changes were made.
func healOptionsConfig(userOpts *OptionsConfig, defaultOpts *OptionsConfig) bool {
	// Check top-level fields
	if userOpts.Editor == "" {
		userOpts.Editor = defaultOpts.Editor
	}
	if userOpts.Theme == "" {
		userOpts.Theme = defaultOpts.Theme
	}

	// Check Discord options
	if userOpts.Discord.Token == "" {
		userOpts.Discord.Token = defaultOpts.Discord.Token
	}
	if userOpts.Discord.ServerID == "" {
		userOpts.Discord.ServerID = defaultOpts.Discord.ServerID
	}
	if userOpts.Discord.DebugChannelID == "" {
		userOpts.Discord.DebugChannelID = defaultOpts.Discord.DebugChannelID
	}

	// Server Map
	_, serverMapErr := LoadServerMapConfig()
	if serverMapErr != nil {
		if os.IsNotExist(serverMapErr) {
			fmt.Println("Creating default server-map.json...")
			if err := SaveServerMapConfig(DefaultServerMapConfig()); err != nil {
				ui.PrintInfo("failed to save default server-map.json")
				return false
			}
		} else {
			ui.PrintInfo("failed to load server-map.json")
			return false
		}
	}

	return true
}

// LogFile returns a file handle to the dex-cli log file.
func LogFile() (*os.File, error) {
	logPath, err := ExpandPath(filepath.Join(DexterRoot, "logs", "dex-cli.log"))
	if err != nil {
		return nil, fmt.Errorf("failed to expand log file path: %w", err)
	}

	// Ensure the directory exists.
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open the file in append mode, create it if it doesn't exist.
	file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	return file, nil
}

// Log writes a message to the dex-cli log file.
func Log(message string) {
	f, err := LogFile()
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()
	_, _ = fmt.Fprintln(f, message)
}

// IsDevMode checks if the EasterCompany source directory exists.
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
