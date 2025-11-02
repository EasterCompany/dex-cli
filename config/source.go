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

	// Check for any dex-*-service directories
	matches, err := filepath.Glob(filepath.Join(easterCompanyPath, "dex-*-service"))
	if err != nil {
		return false
	}

	return len(matches) > 0
}
