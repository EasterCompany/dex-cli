// cmd/utils.go
package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/EasterCompany/dex-cli/config"
)

// FormatBytes converts int64 bytes to a human-readable string.
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// HasArtifacts checks if any file system artifacts for a service exist.
// This includes logs, systemd files, binaries, or config entries.
func HasArtifacts(def config.ServiceDefinition, serviceMap *config.ServiceMapConfig) (bool, error) {
	// 1. Check for log file
	logPath, err := config.ExpandPath(def.GetLogPath())
	if err != nil {
		return false, err
	}
	if checkFileExists(logPath) {
		return true, nil
	}

	// 2. Check for systemd file
	servicePath, err := config.ExpandPath(def.GetSystemdPath())
	if err != nil {
		return false, err
	}
	if checkFileExists(servicePath) {
		return true, nil
	}

	// 3. Check for binary
	binPath, err := config.ExpandPath(def.GetBinaryPath())
	if err != nil {
		return false, err
	}
	if checkFileExists(binPath) {
		return true, nil
	}

	// 4. Check for source code
	sourcePath, err := config.ExpandPath(def.Source)
	if err != nil {
		return false, err
	}
	if checkFileExists(sourcePath) {
		return true, nil
	}

	// 5. Check if in service-map.json
	if isServiceInMap(def, serviceMap) {
		return true, nil
	}

	return false, nil
}

// checkFileExists is a simple wrapper for os.Stat
func checkFileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// isServiceInMap checks if a service is present in the service-map.json
func isServiceInMap(def config.ServiceDefinition, serviceMap *config.ServiceMapConfig) bool {
	if serviceMap == nil || serviceMap.Services == nil {
		return false
	}
	if s, ok := serviceMap.Services[def.Type]; ok {
		for _, entry := range s {
			if entry.ID == def.ID {
				return true
			}
		}
	}
	return false
}

// parseNumericInput cleans and parses a comma-separated string of numbers
func parseNumericInput(input string) ([]int, error) {
	input = strings.TrimSpace(input)
	input = strings.TrimSuffix(input, ",")
	parts := strings.Split(input, ",")
	var indices []int

	if len(parts) == 0 || (len(parts) == 1 && parts[0] == "") {
		return indices, nil
	}

	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		idx, err := strconv.Atoi(trimmed)
		if err != nil {
			return nil, fmt.Errorf("'%s' is not a valid number", trimmed)
		}
		indices = append(indices, idx)
	}
	return indices, nil
}
