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
	// 1. Get initial versions
	// ---
	allServices := config.GetAllServices()
	oldVersions := make(map[string]string)
	for _, s := range allServices {
		if s.IsBuildable() {
			if s.ShortName == "cli" {
				// For the CLI, the "old" version is the one currently running.
				oldVersions[s.ID] = RunningVersion
			} else {
				oldVersions[s.ID] = getServiceVersion(s)
			}
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
	if err := runUnifiedBuildPipeline(dexCliDef, log, true); err != nil {
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

		if err := runUnifiedBuildPipeline(def, log, false); err != nil {
			return err
		}

		// After a successful build, install the systemd service
		if err := installSystemdService(def); err != nil {
			return err
		}

		ui.PrintSuccess(fmt.Sprintf("Successfully built and installed %s!", def.ShortName))
		ui.PrintInfo(fmt.Sprintf("%s  Previous Version: %s%s", ui.ColorDarkGray, oldVersions[def.ID], ui.ColorReset))
		ui.PrintInfo(fmt.Sprintf("%s  Current Version:  %s%s", ui.ColorDarkGray, getServiceVersion(def), ui.ColorReset))
		servicesBuilt++
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
	time.Sleep(2 * time.Second)

	// Get new versions and print changes
	for _, s := range allServices {
		if s.IsBuildable() {
			oldVersionStr := oldVersions[s.ID]
			newVersionStr := getServiceVersion(s)

			oldVersion, errOld := git.Parse(oldVersionStr)
			newVersion, errNew := git.Parse(newVersionStr)

			var oldVersionDisplay, newVersionDisplay string

			// Format old version
			if errOld != nil {
				oldVersionDisplay = ui.Colorize("N/A", ui.ColorDarkGray)
			} else {
				shortTag := oldVersion.Short()
				if errNew == nil && oldVersion.Compare(newVersion) < 0 {
					shortTag = ui.Colorize(shortTag, ui.ColorBrightRed)
				} else {
					shortTag = ui.Colorize(shortTag, ui.ColorReset) // White
				}

				var branchAndCommit string
				if oldVersion.Branch != "" && oldVersion.Commit != "" {
					branchAndCommit = fmt.Sprintf(".%s.%s", oldVersion.Branch, oldVersion.Commit)
					branchAndCommit = ui.Colorize(branchAndCommit, ui.ColorDarkGray)
				}
				oldVersionDisplay = fmt.Sprintf("%s%s", shortTag, branchAndCommit)
			}

			// Format new version
			if errNew != nil {
				newVersionDisplay = ui.Colorize("N/A", ui.ColorDarkGray)
			} else {
				shortTag := newVersion.Short()
				if errOld == nil && newVersion.Compare(oldVersion) > 0 {
					shortTag = ui.Colorize(shortTag, ui.ColorGreen)
				} else {
					shortTag = ui.Colorize(shortTag, ui.ColorReset) // White
				}

				var branchAndCommit string
				if newVersion.Branch != "" && newVersion.Commit != "" {
					branchAndCommit = fmt.Sprintf(".%s.%s", newVersion.Branch, newVersion.Commit)
					branchAndCommit = ui.Colorize(branchAndCommit, ui.ColorDarkGray)
				}
				newVersionDisplay = fmt.Sprintf("%s%s", shortTag, branchAndCommit)
			}

			greyV := ui.Colorize("v", ui.ColorDarkGray)
			ui.PrintInfo(fmt.Sprintf("[%s] %s %s -> %s", s.ShortName, greyV, oldVersionDisplay, newVersionDisplay))
		}
	}

	ui.PrintSuccess(fmt.Sprintf("Successfully built and installed %d service(s).", servicesBuilt+1)) // +1 for dex-cli

	return nil
}
func runUnifiedBuildPipeline(def config.ServiceDefinition, log func(string), isCli bool) error {
	sourcePath, err := config.ExpandPath(def.Source)
	if err != nil {
		return fmt.Errorf("failed to expand source path for %s: %w", def.ShortName, err)
	}

	if !checkFileExists(sourcePath) {
		ui.PrintWarning(fmt.Sprintf("Skipping %s: source code not found at %s. Run 'dex add' to download & install it.", def.ShortName, sourcePath))
		return nil
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
		return fmt.Errorf("%s 'go mod tidy' failed: %w", def.ShortName, err)
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
		return fmt.Errorf("%s 'go fmt' failed: %w", def.ShortName, err)
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
		return fmt.Errorf("%s 'golangci-lint run' failed: %w", def.ShortName, err)
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
		return fmt.Errorf("%s 'go test' failed: %w", def.ShortName, err)
	}

	// ---
	// 5. Build
	// ---
	ui.PrintInfo("Building...")
	outputPath, err := config.ExpandPath(def.GetBinaryPath())
	if err != nil {
		return fmt.Errorf("could not expand binary path for %s: %w", def.ShortName, err)
	}

	var buildCmd *exec.Cmd
	if isCli {
		// Special handling for dex-cli to embed version info
		latestTag, err := git.GetLatestTag(sourcePath)
		if err != nil {
			return fmt.Errorf("failed to get latest git tag for cli: %w", err)
		}
		major, minor, patch, err := git.ParseVersionTag(latestTag)
		if err != nil {
			ui.PrintWarning(fmt.Sprintf("Could not parse tag '%s', defaulting to v0.0.0. Error: %v", latestTag, err))
			major, minor, patch = 0, 0, 0
		}
		newVersion := fmt.Sprintf("v%d.%d.%d", major, minor, patch+1)

		branch, commit := git.GetVersionInfo(sourcePath)
		buildDate := time.Now().UTC().Format("2006-01-02T15:04:05Z")
		buildYear := time.Now().UTC().Format("2006")
		buildHash := "local" // Placeholder for build hash

		ldflags := fmt.Sprintf("-X main.version=%s -X main.branch=%s -X main.commit=%s -X main.buildDate=%s -X main.buildYear=%s -X main.buildHash=%s",
			newVersion, branch, commit, buildDate, buildYear, buildHash)

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
		return fmt.Errorf("%s 'go build' failed: %w", def.ShortName, err)
	}

	return nil
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
