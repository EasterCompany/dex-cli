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

// getServiceVersion gets the current version for a service from its git tags
func getServiceVersion(def config.ServiceDefinition) (major, minor, patch int, err error) {
	sourcePath, expandErr := config.ExpandPath(def.Source)
	if expandErr != nil {
		return 0, 0, 0, fmt.Errorf("failed to expand source path: %w", expandErr)
	}

	// Get latest tag from the service's git repo
	tag, tagErr := git.GetLatestTag(sourcePath)
	if tagErr != nil {
		// No tags found, start at 0.0.0
		return 0, 0, 0, nil
	}

	// Parse the tag
	parsedVer, parseErr := git.Parse(tag)
	if parseErr != nil {
		// Invalid tag format, start at 0.0.0
		return 0, 0, 0, nil
	}

	major, _ = strconv.Atoi(parsedVer.Major)
	minor, _ = strconv.Atoi(parsedVer.Minor)
	patch, _ = strconv.Atoi(parsedVer.Patch)
	return major, minor, patch, nil
}

// hasUncommittedChanges checks if a service has uncommitted changes
func hasUncommittedChanges(def config.ServiceDefinition) bool {
	sourcePath, err := config.ExpandPath(def.Source)
	if err != nil {
		return false
	}

	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = sourcePath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}

	return strings.TrimSpace(string(output)) != ""
}

// getHighestMajorMinor returns the highest major.minor version across all buildable services
func getHighestMajorMinor(services []config.ServiceDefinition) (int, int, error) {
	maxMajor := 0
	maxMinor := 0

	for _, s := range services {
		if !s.IsBuildable() {
			continue
		}

		major, minor, _, err := getServiceVersion(s)
		if err != nil {
			return 0, 0, err
		}

		if major > maxMajor || (major == maxMajor && minor > maxMinor) {
			maxMajor = major
			maxMinor = minor
		}
	}

	return maxMajor, maxMinor, nil
}

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

	var requestedIncrement string
	if len(args) == 1 {
		requestedIncrement = args[0]
		if requestedIncrement != "major" && requestedIncrement != "minor" && requestedIncrement != "patch" {
			return fmt.Errorf("invalid argument '%s': must be 'major', 'minor', or 'patch'", requestedIncrement)
		}
	} else {
		requestedIncrement = "auto"
	}

	log("Build command called...")
	ui.PrintHeader("Building All Services from Local Source")
	allServices := config.GetAllServices()

	// ---
	// THE LAW OF VERSION: Determine versioning strategy
	// ---
	var servicesWithChanges []config.ServiceDefinition
	for _, s := range allServices {
		if s.IsBuildable() && hasUncommittedChanges(s) {
			servicesWithChanges = append(servicesWithChanges, s)
		}
	}

	var incrementType string
	var buildAllServices bool
	var targetMajorAll, targetMinorAll, targetPatchAll int

	switch requestedIncrement {
	case "major":
		// LAW 3: Major increment - force ALL services to same major version
		ui.PrintInfo("Major release: incrementing ALL services to same major version")
		highestMajor, _, err := getHighestMajorMinor(allServices)
		if err != nil {
			return err
		}
		targetMajorAll = highestMajor + 1
		targetMinorAll = 0
		targetPatchAll = 0
		incrementType = "major"
		buildAllServices = true

	case "minor":
		// LAW 2: Minor increment - force ALL services to same minor version
		ui.PrintInfo("Minor release: incrementing ALL services to same minor version")
		highestMajor, highestMinor, err := getHighestMajorMinor(allServices)
		if err != nil {
			return err
		}
		targetMajorAll = highestMajor
		targetMinorAll = highestMinor + 1
		targetPatchAll = 0
		incrementType = "minor"
		buildAllServices = true

	case "patch", "auto":
		// LAW 1: Any repo with changes increments only its own patch
		if len(servicesWithChanges) == 0 {
			ui.PrintWarning("No uncommitted changes detected in any service")
			return nil
		}

		if len(servicesWithChanges) == 1 {
			ui.PrintInfo(fmt.Sprintf("Building %s with patch increment", servicesWithChanges[0].ShortName))
		} else {
			ui.PrintInfo(fmt.Sprintf("Building %d services with individual patch increments", len(servicesWithChanges)))
		}

		incrementType = "patch"
		buildAllServices = false
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
	// 2. Build services based on versioning strategy
	// ---
	type buildTask struct {
		service                               config.ServiceDefinition
		targetMajor, targetMinor, targetPatch int
	}

	var buildTasks []buildTask

	// Determine which services to build and their target versions
	for _, s := range allServices {
		if !s.IsBuildable() {
			continue
		}

		// Skip if not building all services and this service has no changes
		if !buildAllServices {
			hasChanges := false
			for _, changed := range servicesWithChanges {
				if changed.ID == s.ID {
					hasChanges = true
					break
				}
			}
			if !hasChanges {
				continue
			}
		}

		var targetMajor, targetMinor, targetPatch int

		if buildAllServices {
			// Use the shared version for all services
			targetMajor = targetMajorAll
			targetMinor = targetMinorAll
			targetPatch = targetPatchAll
		} else {
			// Individual patch increment
			baseMajor, baseMinor, basePatch, err := getServiceVersion(s)
			if err != nil {
				return fmt.Errorf("failed to get version for %s: %w", s.ShortName, err)
			}
			targetMajor = baseMajor
			targetMinor = baseMinor
			targetPatch = basePatch + 1
		}

		buildTasks = append(buildTasks, buildTask{
			service:     s,
			targetMajor: targetMajor,
			targetMinor: targetMinor,
			targetPatch: targetPatch,
		})
	}

	// Execute build tasks
	for i, task := range buildTasks {
		if i > 0 {
			fmt.Println()
		}

		s := task.service
		ui.PrintInfo(fmt.Sprintf("%s%s%s", ui.ColorCyan, fmt.Sprintf("# Building %s", s.ShortName), ui.ColorReset))

		baseMajor, baseMinor, basePatch, err := getServiceVersion(s)
		if err != nil {
			return fmt.Errorf("failed to get version for %s: %w", s.ShortName, err)
		}

		ui.PrintInfo(fmt.Sprintf("Incrementing version: %d.%d.%d -> %d.%d.%d (%s)",
			baseMajor, baseMinor, basePatch, task.targetMajor, task.targetMinor, task.targetPatch, incrementType))

		built, buildErr := utils.RunUnifiedBuildPipeline(s, log, task.targetMajor, task.targetMinor, task.targetPatch)
		if buildErr != nil {
			return buildErr
		}

		if built {
			if err := utils.InstallSystemdService(s); err != nil {
				return err
			}
			ui.PrintSuccess(fmt.Sprintf("Successfully built and installed %s!", s.ShortName))

			if s.ShortName == "cli" {
				oldVersion := oldVersions[s.ID]
				ui.PrintInfo(fmt.Sprintf("%s  Previous Version: %s%s", ui.ColorDarkGray, oldVersion, ui.ColorReset))
				ui.PrintInfo(fmt.Sprintf("%s  Current Version:  %s%s", ui.ColorDarkGray, utils.GetFullServiceVersion(s), ui.ColorReset))
			} else {
				parsedOldVersion := utils.ParseServiceVersionFromJSON(oldVersions[s.ID])
				ui.PrintInfo(fmt.Sprintf("%s  Previous Version: %s%s", ui.ColorDarkGray, parsedOldVersion, ui.ColorReset))
				ui.PrintInfo(fmt.Sprintf("%s  Current Version:  %s%s", ui.ColorDarkGray, utils.ParseServiceVersionFromJSON(utils.GetFullServiceVersion(s)), ui.ColorReset))
			}
		}
	}

	// ---
	// 3. Git Add, Commit, Push (only for built services)
	// ---
	fmt.Println()
	ui.PrintInfo(ui.Colorize("Git version control", ui.ColorCyan))

	for _, task := range buildTasks {
		if err := gitAddCommitPush(task.service, incrementType, task.targetMajor, task.targetMinor, task.targetPatch); err != nil {
			return err
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
		releaseScript := fmt.Sprintf("%s/EasterCompany/easter.company/scripts/release_dex-cli.sh", os.Getenv("HOME"))

		// Check if release script exists
		if _, err := os.Stat(releaseScript); err == nil {
			ui.PrintInfo("Running release script...")
			releaseCmd := exec.Command(releaseScript)
			releaseCmd.Stdout = os.Stdout
			releaseCmd.Stderr = os.Stderr
			if err := releaseCmd.Run(); err != nil {
				ui.PrintWarning(fmt.Sprintf("Release script failed: %v", err))
				ui.PrintInfo("Version tags have been created and pushed successfully.")
			} else {
				ui.PrintSuccess("Release published successfully!")
			}
		} else {
			ui.PrintInfo("No release script found, skipping publish step.")
			ui.PrintInfo("Version tags have been created and pushed successfully.")
		}
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

	// Create and push tag for ALL builds (to track version history)
	tagName := fmt.Sprintf("%d.%d.%d", major, minor, patch)
	ui.PrintInfo(fmt.Sprintf("[%s] Creating tag %s...", def.ShortName, tagName))

	// Create tag
	tagCmd := exec.Command("git", "tag", tagName)
	tagCmd.Dir = sourcePath
	if output, err := tagCmd.CombinedOutput(); err != nil {
		// If tag already exists, that's okay
		if !strings.Contains(string(output), "already exists") {
			return fmt.Errorf("git tag failed for %s:\n%s", def.ShortName, string(output))
		}
		ui.PrintWarning(fmt.Sprintf("[%s] Tag %s already exists", def.ShortName, tagName))
	} else {
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
