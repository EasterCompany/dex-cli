package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/git"
	"github.com/EasterCompany/dex-cli/ui"
	"github.com/EasterCompany/dex-cli/utils"
)

var RunningVersion string

func Build(args []string) error {
	logFile, err := config.LogFile()
	if err != nil {
		return fmt.Errorf("failed to get log file: %w", err)
	}
	defer func() { _ = logFile.Close() }()

	log := func(message string) {
		_, _ = fmt.Fprintln(logFile, message)
	}

	// Validate arguments
	if len(args) > 1 {
		return fmt.Errorf("build command accepts at most 1 argument (major, minor, or patch)")
	}

	var incrementType string
	if len(args) == 1 {
		incrementType = args[0]
		if incrementType != "major" && incrementType != "minor" && incrementType != "patch" {
			return fmt.Errorf("invalid argument '%s': must be 'major', 'minor', or 'patch'", incrementType)
		}
	}

	// Fetch current version from URL
	const versionURL = "https://easter.company/static/bin/dex-cli/latest.txt"
	var baseMajor, baseMinor, basePatch int

	versionStr, err := git.FetchLatestVersionFromURL(versionURL)
	if err != nil {
		if incrementType != "" {
			// If increment was requested, version fetch failure is fatal
			return fmt.Errorf("failed to fetch version for increment: %w", err)
		}
		// No increment requested, default to 0.0.0
		ui.PrintWarning(fmt.Sprintf("Could not fetch version from %s, defaulting to 0.0.0", versionURL))
		baseMajor, baseMinor, basePatch = 0, 0, 0
	} else {
		// Parse the fetched version
		parsedVer, parseErr := git.Parse(versionStr)
		if parseErr != nil {
			if incrementType != "" {
				return fmt.Errorf("failed to parse version '%s': %w", versionStr, parseErr)
			}
			ui.PrintWarning(fmt.Sprintf("Could not parse version '%s', defaulting to 0.0.0", versionStr))
			baseMajor, baseMinor, basePatch = 0, 0, 0
		} else {
			baseMajor, _ = strconv.Atoi(parsedVer.Major)
			baseMinor, _ = strconv.Atoi(parsedVer.Minor)
			basePatch, _ = strconv.Atoi(parsedVer.Patch)
		}
	}

	// Apply increment if requested
	targetMajor, targetMinor, targetPatch := baseMajor, baseMinor, basePatch
	if incrementType != "" {
		targetMajor, targetMinor, targetPatch, err = git.IncrementVersion(baseMajor, baseMinor, basePatch, incrementType)
		if err != nil {
			return err
		}
		ui.PrintInfo(fmt.Sprintf("Incrementing version: %d.%d.%d -> %d.%d.%d (%s)",
			baseMajor, baseMinor, basePatch, targetMajor, targetMinor, targetPatch, incrementType))
	} else {
		ui.PrintInfo(fmt.Sprintf("Using version: %d.%d.%d (no increment)", targetMajor, targetMinor, targetPatch))
	}

	log("Build command called...")
	ui.PrintHeader("Building All Services from Local Source")
	allServices := config.GetAllServices()

	// Find dex-cli definition
	var dexCliDef config.ServiceDefinition
	for _, s := range allServices {
		if s.ShortName == "cli" {
			dexCliDef = s
			break
		}
	}

	// ---
	// 1. Get initial versions and sizes
	// ---
	oldVersions := make(map[string]string)
	oldSizes := make(map[string]int64)
	for _, s := range allServices {
		if s.IsBuildable() {
			if s.ShortName == "cli" {
				oldVersions[s.ID] = RunningVersion
			} else {
				oldVersions[s.ID] = utils.GetServiceVersion(s)
			}
			oldSizes[s.ID] = utils.GetBinarySize(s)
		}
	}

	// ---
	// 2. Process cli FIRST
	// ---
	oldCliVersion := utils.GetFullServiceVersion(dexCliDef)
	ui.PrintInfo(fmt.Sprintf("%s%s%s", ui.ColorCyan, "# Building cli", ui.ColorReset))
	if _, err := utils.RunUnifiedBuildPipeline(dexCliDef, log, targetMajor, targetMinor, targetPatch); err != nil {
		return err
	}
	ui.PrintSuccess(fmt.Sprintf("Successfully built and installed %s!", dexCliDef.ShortName))
	ui.PrintInfo(fmt.Sprintf("%s  Previous Version: %s%s", ui.ColorDarkGray, oldCliVersion, ui.ColorReset))
	ui.PrintInfo(fmt.Sprintf("%s  Current Version:  %s%s", ui.ColorDarkGray, utils.GetFullServiceVersion(dexCliDef), ui.ColorReset))

	// ---
	// 3. Process all OTHER services
	// ---
	for _, def := range allServices {
		if !def.IsManageable() || def.ShortName == "cli" {
			continue
		}

		fmt.Println()
		ui.PrintInfo(fmt.Sprintf("%s%s%s", ui.ColorCyan, fmt.Sprintf("# Building %s", def.ShortName), ui.ColorReset))

		built, err := utils.RunUnifiedBuildPipeline(def, log, targetMajor, targetMinor, targetPatch)
		if err != nil {
			return err
		}

		if built {
			if err := utils.InstallSystemdService(def); err != nil {
				return err
			}
			ui.PrintSuccess(fmt.Sprintf("Successfully built and installed %s!", def.ShortName))
			parsedOldVersion := utils.ParseServiceVersionFromJSON(oldVersions[def.ID])
			ui.PrintInfo(fmt.Sprintf("%s  Previous Version: %s%s", ui.ColorDarkGray, parsedOldVersion, ui.ColorReset))
			ui.PrintInfo(fmt.Sprintf("%s  Current Version:  %s%s", ui.ColorDarkGray, utils.ParseServiceVersionFromJSON(utils.GetFullServiceVersion(def)), ui.ColorReset))
		}
	}

	// ---
	// 4. Git Add, Commit, Push
	// ---
	fmt.Println()
	ui.PrintInfo(ui.Colorize("Git version control", ui.ColorCyan))
	serviceMap, err := config.LoadServiceMapConfig()
	if err != nil {
		return fmt.Errorf("failed to load service-map.json: %w", err)
	}

	for _, serviceList := range serviceMap.Services {
		for _, serviceEntry := range serviceList {
			def := config.GetServiceDefinition(serviceEntry.ID)
			if def.Type == "os" {
				continue
			}
			if err := gitAddCommitPush(def, incrementType, targetMajor, targetMinor, targetPatch); err != nil {
				return err
			}
		}
	}

	// ---
	// 5. Final Summary
	// ---
	fmt.Println()
	ui.PrintHeader("Complete")
	time.Sleep(2 * time.Second)

	var summaryData []utils.SummaryInfo
	for _, s := range allServices {
		if s.IsBuildable() {
			oldVersionStr := oldVersions[s.ID]
			newVersionStr := utils.GetServiceVersion(s)

			if s.Type != "cli" {
				oldVersionStr = utils.ParseServiceVersionFromJSON(oldVersionStr)
				newVersionStr = utils.ParseServiceVersionFromJSON(newVersionStr)
			}

			// Get the latest commit message from the repository (not from the embedded version)
			// This ensures we show the commit message from any automated commits made during the build
			var commitNote string
			repoPath, err := config.ExpandPath(s.Source)
			if err == nil {
				_, latestCommit := git.GetVersionInfo(repoPath)
				if latestCommit != "" && latestCommit != "unknown" {
					commitNote, _ = git.GetCommitMessage(repoPath, latestCommit)
				} else {
					commitNote = "N/A"
				}
			} else {
				commitNote = "N/A"
			}

			summaryData = append(summaryData, utils.SummaryInfo{
				Service:       s,
				OldVersion:    oldVersionStr,
				NewVersion:    newVersionStr,
				OldSize:       oldSizes[s.ID],
				NewSize:       utils.GetBinarySize(s),
				ChangeSummary: commitNote,
			})
		}
	}

	utils.PrintSummaryTable(summaryData)
	fmt.Println()
	ui.PrintSuccess("All services are built.")

	// ---
	// 6. Run release script if version increment was requested
	// ---
	if incrementType != "" {
		fmt.Println()
		ui.PrintHeader("Publishing Release")
		ui.PrintInfo("Running release script...")
		releaseScript := fmt.Sprintf("%s/EasterCompany/easter.company/scripts/release_dex-cli.sh", os.Getenv("HOME"))
		releaseCmd := exec.Command(releaseScript)
		releaseCmd.Stdout = os.Stdout
		releaseCmd.Stderr = os.Stderr
		if err := releaseCmd.Run(); err != nil {
			return fmt.Errorf("release script failed: %w", err)
		}
		ui.PrintSuccess("Release published successfully!")
	}

	return nil
}

func gitAddCommitPush(def config.ServiceDefinition, incrementType string, major, minor, patch int) error {
	sourcePath, err := config.ExpandPath(def.Source)
	if err != nil {
		return fmt.Errorf("failed to expand source path: %w", err)
	}

	ui.PrintInfo(fmt.Sprintf("[%s] Adding, committing, and pushing changes...", def.ShortName))

	// Add all changes
	addCmd := exec.Command("git", "add", ".")
	addCmd.Dir = sourcePath
	if output, err := addCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git add failed for %s:\n%s", def.ShortName, string(output))
	}

	// Check if there are changes to commit
	statusCmd := exec.Command("git", "status", "--porcelain")
	statusCmd.Dir = sourcePath
	statusOutput, err := statusCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git status failed for %s:\n%s", def.ShortName, string(statusOutput))
	}

	// If no changes, skip commit and push
	if strings.TrimSpace(string(statusOutput)) == "" {
		ui.PrintInfo(fmt.Sprintf("[%s] No changes to commit", def.ShortName))
		return nil
	}

	// Get the diff for commit message generation
	diffCmd := exec.Command("git", "diff", "--cached")
	diffCmd.Dir = sourcePath
	diffOutput, err := diffCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git diff failed for %s:\n%s", def.ShortName, string(diffOutput))
	}

	// Generate commit message using the Ollama model
	commitMsg := utils.GenerateCommitMessage(string(diffOutput))

	// Commit with generated message
	commitCmd := exec.Command("git", "commit", "-m", commitMsg)
	commitCmd.Dir = sourcePath
	if output, err := commitCmd.CombinedOutput(); err != nil {
		if !strings.Contains(string(output), "nothing to commit") {
			return fmt.Errorf("git commit failed for %s:\n%s", def.ShortName, string(output))
		}
	}

	// Push changes
	pushCmd := exec.Command("git", "push")
	pushCmd.Dir = sourcePath
	if output, err := pushCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git push failed for %s:\n%s", def.ShortName, string(output))
	}

	// Create and push tag for minor/major increments
	if incrementType == "minor" || incrementType == "major" {
		tagName := fmt.Sprintf("%d.%d.%d", major, minor, patch)
		ui.PrintInfo(fmt.Sprintf("[%s] Creating tag %s...", def.ShortName, tagName))

		// Create tag
		tagCmd := exec.Command("git", "tag", tagName)
		tagCmd.Dir = sourcePath
		if output, err := tagCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git tag failed for %s:\n%s", def.ShortName, string(output))
		}

		// Push tag
		pushTagCmd := exec.Command("git", "push", "--tags")
		pushTagCmd.Dir = sourcePath
		if output, err := pushTagCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git push --tags failed for %s:\n%s", def.ShortName, string(output))
		}

		ui.PrintSuccess(fmt.Sprintf("[%s] Tag %s created and pushed", def.ShortName, tagName))
	}

	return nil
}
