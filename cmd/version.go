package cmd

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/EasterCompany/dex-cli/ui"
)

func Version(version, branch, commit, buildDate, buildYear string) {
	exePath, err := os.Executable()
	if err != nil {
		exePath = "unknown"
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = ""
	}

	displayPath := "/"
	if homeDir != "" && strings.HasPrefix(exePath, homeDir) {
		displayPath = "~" + strings.TrimPrefix(exePath, homeDir)
	}

	buildID := fmt.Sprintf("%s-%s-%s-%s", version, branch, commit, buildDate)
	arch := fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)

	versionString := fmt.Sprintf(
		"dex-cli @ %s | %s | %s | Easter Company™ © %s",
		displayPath,
		buildID,
		arch,
		buildYear,
	)
	ui.PrintInfo(versionString)
}
