package cmd

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/git"
	"github.com/EasterCompany/dex-cli/ui"
	"github.com/EasterCompany/dex-cli/utils"
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
	if _, err := utils.RunUnifiedBuildPipeline(dexCliDef, log); err != nil {
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

		built, err := utils.RunUnifiedBuildPipeline(def, log)
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
	ui.PrintInfo(ui.Colorize("# Git version control", ui.ColorCyan))
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
			if err := gitAddCommitPush(def); err != nil {
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
		if !s.IsBuildable() {
			oldVersionStr := oldVersions[s.ID]
			newVersionStr := utils.GetServiceVersion(s)

			if s.Type != "cli" {
				oldVersionStr = utils.ParseServiceVersionFromJSON(oldVersionStr)
				newVersionStr = utils.ParseServiceVersionFromJSON(newVersionStr)
			}

			oldVer, _ := git.Parse(oldVersionStr)
			newVer, _ := git.Parse(newVersionStr)

			var changeLog string
			if oldVer != nil && newVer != nil && oldVer.Commit != "" && newVer.Commit != "" {
				repoPath, _ := config.ExpandPath(s.Source)
				changeLog, _ = git.GetCommitLogBetween(repoPath, oldVer.Commit, newVer.Commit)
			}

			summaryData = append(summaryData, utils.SummaryInfo{
				Service:       s,
				OldVersion:    oldVersionStr,
				NewVersion:    newVersionStr,
				OldSize:       oldSizes[s.ID],
				NewSize:       utils.GetBinarySize(s),
				ChangeSummary: changeLog,
			})
		}
	}

	utils.PrintSummaryTable(summaryData)

	fmt.Println()
	ui.PrintSuccess("All services are built.")

	return nil
}

func gitAddCommitPush(def config.ServiceDefinition) error {
	sourcePath, err := config.ExpandPath(def.Source)
	if err != nil {
		return fmt.Errorf("failed to expand source path: %w", err)
	}

	ui.PrintInfo(fmt.Sprintf("[%s] Adding, committing, and pushing changes...", def.ShortName))

	addCmd := exec.Command("git", "add", ".")
	addCmd.Dir = sourcePath
	if output, err := addCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git add failed for %s:\n%s", def.ShortName, string(output))
	}

	commitMsg := "dex build: successful build"
	commitCmd := exec.Command("git", "commit", "-m", commitMsg)
	commitCmd.Dir = sourcePath
	if output, err := commitCmd.CombinedOutput(); err != nil {
		if !strings.Contains(string(output), "nothing to commit") {
			return fmt.Errorf("git commit failed for %s:\n%s", def.ShortName, string(output))
		}
	}

	pushCmd := exec.Command("git", "push")
	pushCmd.Dir = sourcePath
	if output, err := pushCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git push failed for %s:\n%s", def.ShortName, string(output))
	}

	return nil
}
