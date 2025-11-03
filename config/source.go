package config

import (
	"os"
	"path/filepath"
)

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
