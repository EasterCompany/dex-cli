// cmd/pipeline.go
package utils

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/git"
	"github.com/EasterCompany/dex-cli/ui"
)

// gitUpdateService clones or force-pulls a service repository.
func GitUpdateService(def config.ServiceDefinition) error {
	sourcePath, err := config.ExpandPath(def.Source)
	if err != nil {
		return fmt.Errorf("failed to expand source path: %w", err)
	}

	if !CheckFileExists(sourcePath) {
		return fmt.Errorf("source directory not found for %s: %s", def.ShortName, sourcePath)
	}

	// Source exists, force-pull latest changes
	ui.PrintInfo(fmt.Sprintf("[%s] Pulling latest changes...", def.ShortName))

	commands := [][]string{
		{"git", "fetch", "--all"},
		{"git", "reset", "--hard", "origin/main"},
		{"git", "pull", "origin", "main"},
	}

	for _, args := range commands {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = sourcePath
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("command failed: %s\nOutput: %s\nError: %w", strings.Join(args, " "), string(output), err)
		}
	}
	return nil
}

// RunUnifiedBuildPipeline executes the full build pipeline for a service.
func RunUnifiedBuildPipeline(def config.ServiceDefinition, log func(string), major, minor, patch int) (bool, error) {
	sourcePath, err := config.ExpandPath(def.Source)
	if err != nil {
		return false, fmt.Errorf("failed to expand source path for %s: %w", def.ShortName, err)
	}

	if !CheckFileExists(sourcePath) {
		ui.PrintWarning(fmt.Sprintf("Skipping %s: source code not found at %s. Run 'dex add' to download & install it.", def.ShortName, sourcePath))
		return false, nil
	}

	log(fmt.Sprintf("Building %s from local source...", def.ShortName))

	// ---
	// 0. Stop service if running (to avoid "Text file busy" error)
	// ---
	if def.IsManageable() {
		ui.PrintInfo("Stopping service if running...")
		stopCmd := exec.Command("systemctl", "--user", "stop", def.SystemdName)
		_ = stopCmd.Run() // Ignore errors - service might not be running
	}

	// ---
	// 1. Tidy
	// ---
	ui.PrintInfo("Ensuring Go modules are tidy...")
	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = sourcePath
	tidyCmd.Stdout = os.Stdout
	tidyCmd.Stderr = os.Stderr
	if err := tidyCmd.Run(); err != nil {
		log(fmt.Sprintf("%s 'go mod tidy' failed: %v", def.ShortName, err))
		return false, fmt.Errorf("%s 'go mod tidy' failed: %w", def.ShortName, err)
	}

	// ---
	// 2. Format
	// ---
	ui.PrintInfo("Formatting...")
	formatCmd := exec.Command("go", "fmt", "./...")
	formatCmd.Dir = sourcePath
	formatCmd.Stdout = os.Stdout
	formatCmd.Stderr = os.Stderr
	if err := formatCmd.Run(); err != nil {
		log(fmt.Sprintf("%s 'go fmt' failed: %v", def.ShortName, err))
		return false, fmt.Errorf("%s 'go fmt' failed: %w", def.ShortName, err)
	}

	// ---
	// 3. Lint
	// ---
	ui.PrintInfo("Linting...")
	lintCmd := exec.Command("golangci-lint", "run")
	lintCmd.Dir = sourcePath
	lintCmd.Stdout = os.Stdout
	lintCmd.Stderr = os.Stderr
	if err := lintCmd.Run(); err != nil {
		log(fmt.Sprintf("%s 'golangci-lint run' failed: %v", def.ShortName, err))
		return false, fmt.Errorf("%s 'golangci-lint run' failed: %w", def.ShortName, err)
	}

	// ---
	// 4. Test
	// ---
	ui.PrintInfo("Testing...")
	testCmd := exec.Command("go", "test", "-v", "./...")
	testCmd.Dir = sourcePath
	testCmd.Stdout = os.Stdout
	testCmd.Stderr = os.Stderr
	if err := testCmd.Run(); err != nil {
		log(fmt.Sprintf("%s 'go test' failed: %v", def.ShortName, err))
		return false, fmt.Errorf("%s 'go test' failed: %w", def.ShortName, err)
	}

	// ---
	// 5. Build
	// ---
	ui.PrintInfo("Building...")
	outputPath, err := config.ExpandPath(def.GetBinaryPath())
	if err != nil {
		return false, fmt.Errorf("could not expand binary path for %s: %w", def.ShortName, err)
	}

	var buildCmd *exec.Cmd
	// Use the provided version (major, minor, patch) instead of calculating from tags
	branch, commit := git.GetVersionInfo(sourcePath)
	buildDate := time.Now().UTC().Format("2006-01-02-15-04-05") // Hyphenated format
	buildYear := time.Now().UTC().Format("2006")
	buildArch := "linux-amd64"         // Hyphenated format
	buildHash := GenerateRandomHash(8) // Generate an 8-character random hash

	// Format the version string to match the new parsing logic (M.m.p.branch.commit.date.arch.hash)
	fullVersionStr := fmt.Sprintf("%d.%d.%d.%s.%s.%s.%s.%s",
		major, minor, patch, branch, commit, buildDate, buildArch, buildHash)

	// Check if service has a Makefile (new universal standard)
	makefilePath := filepath.Join(sourcePath, "Makefile")
	if _, err := os.Stat(makefilePath); err == nil {
		// Makefile exists - use it for building
		log(fmt.Sprintf("%s Using Makefile for build", def.ShortName))

		// Clean before building to ensure fresh compilation
		cleanCmd := exec.Command("make", "clean")
		cleanCmd.Dir = sourcePath
		cleanCmd.Stdout = os.Stdout
		cleanCmd.Stderr = os.Stderr
		if err := cleanCmd.Run(); err != nil {
			log(fmt.Sprintf("%s 'make clean' failed: %v", def.ShortName, err))
			return false, fmt.Errorf("%s 'make clean' failed: %w", def.ShortName, err)
		}

		// Export build variables for Makefile to use
		ldflags := fmt.Sprintf("-X main.version=%s -X main.branch=%s -X main.commit=%s -X main.buildDate=%s -X main.buildYear=%s -X main.buildHash=%s -X main.arch=%s",
			fullVersionStr, branch, commit, buildDate, buildYear, buildHash, buildArch)

		buildCmd = exec.Command("make", "install")
		buildCmd.Dir = sourcePath
		buildCmd.Env = append(os.Environ(),
			fmt.Sprintf("LDFLAGS=%s", ldflags),
			fmt.Sprintf("VERSION=%s", fullVersionStr),
			fmt.Sprintf("BUILD_DATE=%s", buildDate),
		)
		buildCmd.Stdout = os.Stdout
		buildCmd.Stderr = os.Stderr
		if err := buildCmd.Run(); err != nil {
			log(fmt.Sprintf("%s 'make install' failed: %v", def.ShortName, err))
			return false, fmt.Errorf("%s 'make install' failed: %w", def.ShortName, err)
		}
	} else {
		// No Makefile - use standard Go build
		log(fmt.Sprintf("%s Using standard go build", def.ShortName))
		ldflags := fmt.Sprintf("-X main.version=%s -X main.branch=%s -X main.commit=%s -X main.buildDate=%s -X main.buildYear=%s -X main.buildHash=%s -X main.arch=%s",
			fullVersionStr, branch, commit, buildDate, buildYear, buildHash, buildArch)
		buildCmd = exec.Command("go", "build", "-ldflags", ldflags, "-o", outputPath, ".")
		buildCmd.Dir = sourcePath
		buildCmd.Stdout = os.Stdout
		buildCmd.Stderr = os.Stderr
		if err := buildCmd.Run(); err != nil {
			log(fmt.Sprintf("%s 'go build' failed: %v", def.ShortName, err))
			return false, fmt.Errorf("%s 'go build' failed: %w", def.ShortName, err)
		}
	}

	return true, nil
}

// InstallSystemdService creates and enables the systemd service file.
func InstallSystemdService(def config.ServiceDefinition) error {
	if !def.IsManageable() {
		return nil // Skip for cli, os
	}

	ui.PrintInfo(fmt.Sprintf("[%s] Installing systemd service...", def.ShortName))

	serviceFilePath, err := config.ExpandPath(def.GetSystemdPath())
	if err != nil {
		return fmt.Errorf("could not get systemd path: %w", err)
	}

	executablePath, err := config.ExpandPath(def.GetBinaryPath())
	if err != nil {
		return fmt.Errorf("could not expand binary path: %w", err)
	}

	logPath, err := config.ExpandPath(def.GetLogPath())
	if err != nil {
		return fmt.Errorf("could not expand log path: %w", err)
	}

	// Ensure directories exist
	if err := os.MkdirAll(filepath.Dir(serviceFilePath), 0755); err != nil {
		return fmt.Errorf("could not create systemd directory: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		return fmt.Errorf("could not create log directory: %w", err)
	}

	var execStartCommand string
	if def.Type == "fe" {
		// For frontend services, execute `dex serve`
		// GetServiceDefinition returns a value, so take its address to call pointer method GetBinaryPath
		cliServiceDef := config.GetServiceDefinition("dex-cli")
		dexBinaryPath, err := config.ExpandPath((&cliServiceDef).GetBinaryPath())
		if err != nil {
			return fmt.Errorf("could not get dex-cli binary path: %w", err)
		}
		sourcePath, err := config.ExpandPath(def.Source)
		if err != nil {
			return fmt.Errorf("could not expand source path for %s: %w", def.ShortName, err)
		}
		execStartCommand = fmt.Sprintf("%s serve --dir %s --port %s", dexBinaryPath, sourcePath, def.Port)
	} else {
		// For other services, execute their own binary
		execStartCommand = executablePath
	}

	// Create service file content
	serviceFileContent := fmt.Sprintf(`[Unit]
Description=%s
After=network.target

[Service]
ExecStart=%s
Restart=always
RestartSec=3
StandardOutput=append:%s
StandardError=append:%s

[Install]
WantedBy=default.target
`, def.ID, execStartCommand, logPath, logPath)

	// Write the service file
	if err := os.WriteFile(serviceFilePath, []byte(serviceFileContent), 0644); err != nil {
		return fmt.Errorf("could not write service file: %w", err)
	}

	// Run systemctl commands
	commands := [][]string{
		{"systemctl", "--user", "daemon-reload"},
		{"systemctl", "--user", "enable", def.SystemdName},
		{"systemctl", "--user", "restart", def.SystemdName},
	}

	for _, args := range commands {
		cmd := exec.Command(args[0], args[1:]...)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("command failed: %s\nOutput: %s\nError: %w", strings.Join(args, " "), string(output), err)
		}
	}
	return nil
}
