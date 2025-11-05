package cmd

import (
	"os"

	"github.com/EasterCompany/dex-cli/config"
)

// getBinarySize returns the size of a service's binary in bytes.
// It returns 0 if the binary does not exist or if an error occurs.
func getBinarySize(def config.ServiceDefinition) int64 {
	binaryPath, err := config.ExpandPath(def.GetBinaryPath())
	if err != nil {
		return 0 // Cannot expand path
	}

	stat, err := os.Stat(binaryPath)
	if err != nil {
		return 0 // File does not exist or other error
	}

	return stat.Size()
}
