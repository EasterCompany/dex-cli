package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/git"
	"github.com/EasterCompany/dex-cli/ui"
)

// Build compiles all services from their local source.
// It runs the full (format, lint, test, build, install) pipeline.
func Build(args []string) error {
	logFile, err := config.LogFile()
	if err != nil {
		return fmt.Errorf("failed to get log file: %w", err)
	}
	defer func() { _ = logFile.Close() }()

	log := func(message string) {
		_, _ = fmt.Fprintln(logFile, message)
	}

	if len(args) > 0 {
		return fmt.Errorf("build command takes no arguments")
	}

	log("Build command called...")
	ui.PrintHeader("Building All Services from Local Source")

	// ---
	// 1. Get initial versions and sizes
	// ---
	allServices := config.GetAllServices()
	oldVersions := make(map[string]string)
	oldSizes := make(map[string]int64)
	for _, s := range allServices {
		if s.IsBuildable() {
			oldVersions[s.ID] = getServiceVersion(s)
			oldSizes[s.ID] = getBinarySize(s)
		}
	}

	var dexCliDef config.ServiceDefinition
	// Find dex-cli definition
	for _, s := range allServices {
		if s.ShortName == "cli" {
			dexCliDef = s
			break
		}
	}

	// ---
	// 2. Process cli FIRST
	// ---
	ui.PrintInfo(fmt.Sprintf("%s%s%s", ui.ColorCyan, "# Building cli", ui.ColorReset))
	if _, err := runUnifiedBuildPipeline(dexCliDef, log, true); err != nil {
		return err
	}

	// ---
	// 3. Process all OTHER services
	// ---
	servicesBuilt := 0
	for _, def := range allServices {
		if !def.IsManageable() || def.ShortName == "cli" {
			continue
		}

		fmt.Println()
		ui.PrintInfo(fmt.Sprintf("%s%s%s", ui.ColorCyan, fmt.Sprintf("# Building %s", def.ShortName), ui.ColorReset))

		built, err := runUnifiedBuildPipeline(def, log, false)
		if err != nil {
			return err
		}

		if built {
			// After a successful build, install the systemd service
			if err := installSystemdService(def); err != nil {
				return err
			}

			ui.PrintSuccess(fmt.Sprintf("Successfully built and installed %s!", def.ShortName))
			ui.PrintInfo(fmt.Sprintf("%s  Previous Version: %s%s", ui.ColorDarkGray, oldVersions[def.ID], ui.ColorReset))
			ui.PrintInfo(fmt.Sprintf("%s  Current Version:  %s%s", ui.ColorDarkGray, getServiceVersion(def), ui.ColorReset))
			servicesBuilt++
		}
	}

	// ---
	// 4. Git Add, Commit, Push
	// ---
	fmt.Println()
	ui.PrintInfo(ui.Colorize("# Git version control", ui.ColorCyan))
	serviceMap, err := config.LoadServiceMapConfig()
	if err != nil {
		return fmt.Errorf("failed to load service-map.json: %w", err)
	}

	for _, serviceList := range serviceMap.Services {
		for _, serviceEntry := range serviceList {
			def := config.GetServiceDefinition(serviceEntry.ID)
			// Skip services of type "os" as they don't have git repositories
			if def.Type == "os" {
				continue
			}
			if err := gitAddCommitPush(def); err != nil {
				return err // Stop-on-failure
			}
		}
	}

	// ---
	// 5. Final Summary
	// ---
	fmt.Println()
	ui.PrintHeader("Complete")

	// Add a small delay to allow services to restart
	time.Sleep(5 * time.Second)

	// Get new versions and print changes for ALL services
	for _, s := range allServices {
		if s.IsBuildable() {
			oldVersionStr := oldVersions[s.ID]
			newVersionStr := getServiceVersion(s)
			oldSize := oldSizes[s.ID]
			newSize := getBinarySize(s)
			ui.PrintInfo(formatSummaryLine(s, oldVersionStr, newVersionStr, oldSize, newSize))
		}
	}

	ui.PrintSuccess(fmt.Sprintf("Successfully built and installed %d service(s).", servicesBuilt+1)) // +1 for dex-cli

	return nil
}

func runUnifiedBuildPipeline(def config.ServiceDefinition, log func(string), isCli bool) (bool, error) {
	sourcePath, err := config.ExpandPath(def.Source)
	if err != nil {
		return false, fmt.Errorf("failed to expand source path for %s: %w", def.ShortName, err)
	}

	if !checkFileExists(sourcePath) {
		ui.PrintWarning(fmt.Sprintf("Skipping %s: source code not found at %s. Run 'dex add' to download & install it.", def.ShortName, sourcePath))
		return false, nil
	}

	log(fmt.Sprintf("Building %s from local source...", def.ShortName))

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
	if isCli {
		// Special handling for dex-cli to embed version info
		latestTag, err := git.GetLatestTag(sourcePath)
		if err != nil {
			return false, fmt.Errorf("failed to get latest git tag for cli: %w", err)
		}
		major, minor, patch, err := git.ParseVersionTag(latestTag)
		if err != nil {
			ui.PrintWarning(fmt.Sprintf("Could not parse tag '%s', defaulting to 0.0.0. Error: %v", latestTag, err))
			major, minor, patch = 0, 0, 0
		}

		branch, commit := git.GetVersionInfo(sourcePath)
		buildDate := time.Now().UTC().Format("2006-01-02T15:04:05Z")
		buildYear := time.Now().UTC().Format("2006")
		buildArch := "linux/amd64"
		buildHash := "local" // Placeholder for build hash

		// Format the version string to match the new parsing logic (M.m.p.branch.commit.date.arch.hash)
		fullVersionStr := fmt.Sprintf("%d.%d.%d.%s.%s.%s.%s.%s",
			major, minor, patch+1, branch, commit, buildDate, buildArch, buildHash)

		ldflags := fmt.Sprintf("-X main.version=%s -X main.branch=%s -X main.commit=%s -X main.buildDate=%s -X main.buildYear=%s -X main.buildHash=%s",
			fullVersionStr, branch, commit, buildDate, buildYear, buildHash)
		buildCmd = exec.Command("go", "build", "-ldflags", ldflags, "-o", outputPath, ".")
	} else {
		// Standard build for other services
		buildCmd = exec.Command("go", "build", "-o", outputPath, ".")
	}

	buildCmd.Dir = sourcePath
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	if err := buildCmd.Run(); err != nil {
		log(fmt.Sprintf("%s 'go build' failed: %v", def.ShortName, err))
		return false, fmt.Errorf("%s 'go build' failed: %w", def.ShortName, err)
	}

	return true, nil
}

func gitAddCommitPush(def config.ServiceDefinition) error {
	sourcePath, err := config.ExpandPath(def.Source)
	if err != nil {
		return fmt.Errorf("failed to expand source path: %w", err)
	}

	ui.PrintInfo(fmt.Sprintf("[%s] Adding, committing, and pushing changes...", def.ShortName))

	// Git Add
	addCmd := exec.Command("git", "add", ".")
	addCmd.Dir = sourcePath
	if output, err := addCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git add failed for %s:\n%s", def.ShortName, string(output))
	}

	// Git Commit
	commitMsg := "dex build: successful build"
	commitCmd := exec.Command("git", "commit", "-m", commitMsg)
	commitCmd.Dir = sourcePath
	if output, err := commitCmd.CombinedOutput(); err != nil {
		// It's possible there are no changes to commit, so we can ignore this error
		if !strings.Contains(string(output), "nothing to commit") {
			return fmt.Errorf("git commit failed for %s:\n%s", def.ShortName, string(output))
		}
	}

	// Git Push
	pushCmd := exec.Command("git", "push")
	pushCmd.Dir = sourcePath
	if output, err := pushCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git push failed for %s:\n%s", def.ShortName, string(output))
	}

	return nil
}
