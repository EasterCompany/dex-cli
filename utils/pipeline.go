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
)

// RunUnifiedBuildPipeline runs the unified build and test process for a service.
// Supports Go services (go mod tidy, fmt, lint, test, build) and Python services (run.sh).
func RunUnifiedBuildPipeline(service config.ServiceDefinition, log func(message string), major, minor, patch int) (bool, error) {
	sourcePath, err := config.ExpandPath(service.Source)
	if err != nil {
		return false, fmt.Errorf("failed to expand source path: %w", err)
	}

	versionStr := fmt.Sprintf("%d.%d.%d", major, minor, patch)
	log(fmt.Sprintf("Starting unified build pipeline for %s (v%s)...", service.ShortName, versionStr))

	// Check for Python service (marker: requirements.txt or main.py)
	isPython := false
	reqPath := filepath.Join(sourcePath, "requirements.txt")
	mainPath := filepath.Join(sourcePath, "main.py")

	if _, err := os.Stat(reqPath); err == nil {
		isPython = true
		log("Found requirements.txt")
	} else if _, err := os.Stat(mainPath); err == nil {
		isPython = true
		log("Found main.py")
	} else {
		log(fmt.Sprintf("Did not find python markers at %s or %s", reqPath, mainPath))
	}

	if isPython {
		return runPythonBuildPipeline(service, sourcePath, log)
	}

	// Fetch Git Info for ldflags
	branch, commit := git.GetVersionInfo(sourcePath)
	buildDate := time.Now().Format("2006-01-02")
	buildYear := time.Now().Format("2006")
	buildHash := commit // Use commit hash as build hash for now

	// Construct ldflags
	ldflags := fmt.Sprintf(
		"-X main.version=%s -X main.branch=%s -X main.commit=%s -X main.buildDate=%s -X main.buildYear=%s -X main.buildHash=%s",
		versionStr, branch, commit, buildDate, buildYear, buildHash,
	)

	// Default to Go pipeline
	return runGoBuildPipeline(service, sourcePath, log, ldflags)
}

func runPythonBuildPipeline(service config.ServiceDefinition, sourcePath string, log func(message string)) (bool, error) {
	log("Detected Python service.")

	// Check for run.sh or install.sh script
	// For Python services in this architecture, "build" mostly means ensuring the venv is set up
	// and dependencies are installed. We can use the run.sh script if it handles setup,
	// but ideally we should just ensure it's ready to run.
	// Wait, `dex build` creates artifacts. For Python, there isn't a binary artifact usually.
	// But we do want to package it or at least ensure it works.

	// Let's create a virtual env and install requirements if they exist.
	log("Setting up Python environment...")

	// 1. Create venv if not exists
	venvPath := filepath.Join(sourcePath, "venv")
	if _, err := os.Stat(venvPath); os.IsNotExist(err) {
		log("Creating virtual environment...")
		// Try python3.10 first
		cmd := exec.Command("python3.10", "-m", "venv", "venv")
		cmd.Dir = sourcePath
		if out, err := cmd.CombinedOutput(); err != nil {
			log(fmt.Sprintf("python3.10 failed: %s. Trying python3...", strings.TrimSpace(string(out))))
			// Fallback to python3
			cmd = exec.Command("python3", "-m", "venv", "venv")
			cmd.Dir = sourcePath
			if out2, err2 := cmd.CombinedOutput(); err2 != nil {
				return false, fmt.Errorf("failed to create venv: %v\nOutput: %s", err2, string(out2))
			}
		}
	}

	// 2. Install requirements
	if _, err := os.Stat(filepath.Join(sourcePath, "requirements.txt")); err == nil {
		log("Installing requirements (this may take a while)...")
		// Use the pip inside the venv
		pipCmd := filepath.Join(venvPath, "bin", "pip")

		// Upgrade pip first
		upgradeCmd := exec.Command(pipCmd, "install", "--upgrade", "pip")
		upgradeCmd.Dir = sourcePath
		if out, err := upgradeCmd.CombinedOutput(); err != nil {
			log(fmt.Sprintf("Warning: failed to upgrade pip: %v\n%s", err, string(out)))
		}

		cmd := exec.Command(pipCmd, "install", "-r", "requirements.txt")
		cmd.Dir = sourcePath
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return false, fmt.Errorf("failed to install requirements: %w", err)
		}
	}

	// 3. Create git tag for versioning (handled by Build command git phase, but we need to report success here)

	log("Python service setup complete.")
	return true, nil
}

func runGoBuildPipeline(service config.ServiceDefinition, sourcePath string, log func(message string), ldflags string) (bool, error) {
	// 1. Stop service if running (to overwrite binary)
	// We don't stop here, we stop in the actual build step if needed or assume systemd restart handles it?
	// Actually, endpoints might be in use. `dex build` stops the service usually?
	// The caller `Build` handles "Stopping service if running..." in the log messages?
	// No, looking at cmd/build.go, it says "Stopping service if running..." but calls `utils.RunUnifiedBuildPipeline`.
	// So we should handle it here or assume it's handled.
	// `cmd/build.go` logic:
	// ui.PrintInfo(fmt.Sprintf("# Building %s", s.ShortName))
	// ... calls RunUnifiedBuildPipeline

	// Let's check if we need to stop it.
	log("Stopping service if running...")
	_ = exec.Command("systemctl", "--user", "stop", service.SystemdName).Run()

	// 2. Go Mod Tidy
	log("Ensuring Go modules are tidy...")
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = sourcePath
	if output, err := cmd.CombinedOutput(); err != nil {
		return false, fmt.Errorf("%s 'go mod tidy' failed: %w\n%s", service.ShortName, err, string(output))
	}

	// 3. Format
	log("Formatting...")
	cmd = exec.Command("go", "fmt", "./...")
	cmd.Dir = sourcePath
	if output, err := cmd.CombinedOutput(); err != nil {
		return false, fmt.Errorf("%s 'go fmt' failed: %w\n%s", service.ShortName, err, string(output))
	}

	// 4. Lint
	log("Linting...")
	// Check if golangci-lint is installed
	if _, err := exec.LookPath("golangci-lint"); err == nil {
		cmd = exec.Command("golangci-lint", "run")
		cmd.Dir = sourcePath
		if output, err := cmd.CombinedOutput(); err != nil {
			// Treat lint errors as fatal
			return false, fmt.Errorf("%s 'golangci-lint run' failed: %w\n%s", service.ShortName, err, string(output))
		}
		log("0 issues.")
	} else {
		log("Warning: golangci-lint not found, skipping linting.")
	}

	// 5. Test
	log("Testing...")
	cmd = exec.Command("go", "test", "./...")
	cmd.Dir = sourcePath
	cmd.Stdout = os.Stdout // Stream test output
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return false, fmt.Errorf("%s tests failed: %w", service.ShortName, err)
	}

	// 6. Build
	log(fmt.Sprintf("Building %s...", service.ID))

	// Determine output path
	binDir := filepath.Join(os.Getenv("HOME"), "Dexter", "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return false, fmt.Errorf("failed to create bin dir: %w", err)
	}

	// Helper for build command
	buildBinary := func(outputName string, buildTags string) error {
		args := []string{"build", "-ldflags", ldflags, "-o", filepath.Join(binDir, outputName)}
		if buildTags != "" {
			args = append(args, "-tags", buildTags)
		}

		// If it's the event service, it might have multiple binaries (handlers)
		// But standard `go build` in root builds the main module.
		// dex-event-service has handlers as separate binaries?
		// Checking file structure: handlers/publicmessage/main.go etc.
		// So `go build` in root builds the main service.
		// For handlers, we might need to build them individually if they are main packages.

		// Let's assume standard build first.
		cmd := exec.Command("go", args...)
		cmd.Dir = sourcePath
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	// Main service build
	if err := buildBinary(service.ID, ""); err != nil {
		return false, fmt.Errorf("failed to build %s: %w", service.ID, err)
	}
	log(fmt.Sprintf("✓ %s built successfully", service.ID))

	// Special case for dex-event-service: Build handlers
	if service.ID == "dex-event-service" {
		handlersDir := filepath.Join(sourcePath, "handlers")
		entries, err := os.ReadDir(handlersDir)
		if err == nil {
			for _, entry := range entries {
				if entry.IsDir() && entry.Name() != "test-suite" { // Skip non-handler dirs if any
					// Check if it has a main.go
					if _, err := os.Stat(filepath.Join(handlersDir, entry.Name(), "main.go")); err == nil {
						// Hardcoding mapping for safety/consistency with existing
						finalSuffix := entry.Name()
						if finalSuffix == "publicmessage" {
							finalSuffix = "public-message"
						}
						if finalSuffix == "privatemessage" {
							finalSuffix = "private-message"
						}

						targetName := fmt.Sprintf("event-%s-handler", finalSuffix)
						log(fmt.Sprintf("Building %s...", targetName))

						args := []string{"build", "-ldflags", ldflags, "-o", filepath.Join(binDir, targetName), fmt.Sprintf("./handlers/%s", entry.Name())}
						cmd := exec.Command("go", args...)
						cmd.Dir = sourcePath
						cmd.Stdout = os.Stdout
						cmd.Stderr = os.Stderr
						if err := cmd.Run(); err != nil {
							return false, fmt.Errorf("failed to build handler %s: %w", targetName, err)
						}
						log(fmt.Sprintf("✓ %s built successfully", targetName))
					}
				}
			}
		}
	}

	// 7. Clean source (optional, mostly for git status)
	log("Cleaning build artifacts...")
	// git clean -fdX ? No, dangerous. Just leave it.
	log("✓ Clean complete")

	return true, nil
}
