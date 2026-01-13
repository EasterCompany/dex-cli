package utils

import (
	"context"
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
func RunUnifiedBuildPipeline(ctx context.Context, service config.ServiceDefinition, log func(message string), major, minor, patch int) (bool, error) {
	sourcePath, err := config.ExpandPath(service.Source)
	if err != nil {
		return false, fmt.Errorf("failed to expand source path: %w", err)
	}

	versionStr := fmt.Sprintf("%d.%d.%d", major, minor, patch)
	log(fmt.Sprintf("Starting unified build pipeline for %s (v%s)...", service.ShortName, versionStr))

	// Check for Go service (prioritize over Python if go.mod exists)
	goModPath := filepath.Join(sourcePath, "go.mod")
	if _, err := os.Stat(goModPath); err == nil {
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

		return runGoBuildPipeline(ctx, service, sourcePath, log, ldflags, versionStr, branch, commit)
	}

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
		return runPythonBuildPipeline(ctx, service, sourcePath, log)
	}

	// Default to Go pipeline (fallback)
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

	return runGoBuildPipeline(ctx, service, sourcePath, log, ldflags, versionStr, branch, commit)
}

func runPythonBuildPipeline(ctx context.Context, service config.ServiceDefinition, sourcePath string, log func(message string)) (bool, error) {
	log("Detected Python service.")

	// Let's create a virtual env and install requirements if they exist.
	log("Setting up Python environment...")

	// 1. Create venv if not exists
	venvPath := filepath.Join(sourcePath, "venv")
	if _, err := os.Stat(venvPath); os.IsNotExist(err) {
		log("Creating virtual environment...")
		cmd := exec.CommandContext(ctx, "python3.10", "-m", "venv", "venv")
		cmd.Dir = sourcePath
		if out, err := cmd.CombinedOutput(); err != nil {
			log(fmt.Sprintf("python3.10 failed: %s. Trying python3.14...", strings.TrimSpace(string(out))))
			cmd = exec.CommandContext(ctx, "python3.14", "-m", "venv", "venv")
			cmd.Dir = sourcePath
			if out2, err2 := cmd.CombinedOutput(); err2 != nil {
				return false, fmt.Errorf("failed to create venv: %v\nOutput: %s", err2, string(out2))
			}
		}
	}

	// 2. Install requirements
	if _, err := os.Stat(filepath.Join(sourcePath, "requirements.txt")); err == nil {
		log("Installing requirements (this may take a while)...")
		pipCmd := filepath.Join(venvPath, "bin", "pip")

		upgradeCmd := exec.CommandContext(ctx, pipCmd, "install", "--upgrade", "pip")
		upgradeCmd.Dir = sourcePath
		if out, err := upgradeCmd.CombinedOutput(); err != nil {
			log(fmt.Sprintf("Warning: failed to upgrade pip: %v\n%s", err, string(out)))
		}

		cmd := exec.CommandContext(ctx, pipCmd, "install", "-r", "requirements.txt")
		cmd.Dir = sourcePath
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return false, fmt.Errorf("failed to install requirements: %w", err)
		}
	}

	log("Python service setup complete.")
	return true, nil
}

func runGoBuildPipeline(ctx context.Context, service config.ServiceDefinition, sourcePath string, log func(message string), ldflags string, versionStr string, branch string, commit string) (bool, error) {
	log("Stopping service if running...")
	_ = exec.CommandContext(ctx, "systemctl", "--user", "stop", service.SystemdName).Run()

	// 2. Go Mod Tidy
	log("Ensuring Go modules are tidy...")
	cmd := exec.CommandContext(ctx, "go", "mod", "tidy")
	cmd.Dir = sourcePath
	if output, err := cmd.CombinedOutput(); err != nil {
		return false, fmt.Errorf("%s 'go mod tidy' failed: %w\n%s", service.ShortName, err, string(output))
	}

	// 3. Format
	log("Formatting...")
	cmd = exec.CommandContext(ctx, "go", "fmt", "./...")
	cmd.Dir = sourcePath
	if output, err := cmd.CombinedOutput(); err != nil {
		return false, fmt.Errorf("%s 'go fmt' failed: %w\n%s", service.ShortName, err, string(output))
	}

	// 4. Lint
	log("Linting...")
	if _, err := exec.LookPath("golangci-lint"); err == nil {
		cmd = exec.CommandContext(ctx, "golangci-lint", "run")
		cmd.Dir = sourcePath
		if output, err := cmd.CombinedOutput(); err != nil {
			return false, fmt.Errorf("%s 'golangci-lint run' failed: %w\n%s", service.ShortName, err, string(output))
		}
		log("0 issues.")
	} else {
		log("Warning: golangci-lint not found, skipping linting.")
	}

	// 5. Test
	log("Testing...")
	cmd = exec.CommandContext(ctx, "go", "test", "./...")
	cmd.Dir = sourcePath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return false, fmt.Errorf("%s tests failed: %w", service.ShortName, err)
	}

	// 6. Build
	log(fmt.Sprintf("Building %s...", service.ID))

	binDir := filepath.Join(os.Getenv("HOME"), "Dexter", "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return false, fmt.Errorf("failed to create bin dir: %w", err)
	}

	buildBinary := func(outputName string, buildTags string) error {
		args := []string{"build", "-ldflags", ldflags, "-o", filepath.Join(binDir, outputName)}
		if buildTags != "" {
			args = append(args, "-tags", buildTags)
		}

		cmd := exec.CommandContext(ctx, "go", args...)
		cmd.Dir = sourcePath
		// Inject environment variables for binaries that need them during build or runtime
		cmd.Env = append(os.Environ(),
			fmt.Sprintf("DEX_VERSION=%s", versionStr),
			fmt.Sprintf("DEX_BRANCH=%s", branch),
			fmt.Sprintf("DEX_COMMIT=%s", commit),
		)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	outputName := service.ID
	if service.ShortName == "cli" {
		outputName = "dex"
	}
	if err := buildBinary(outputName, ""); err != nil {
		return false, fmt.Errorf("failed to build %s: %w", service.ID, err)
	}
	log(fmt.Sprintf("✓ %s built successfully", service.ID))

	log("Cleaning build artifacts...")
	log("✓ Clean complete")

	return true, nil
}
