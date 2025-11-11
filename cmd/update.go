package cmd

import (
	"fmt"
	"time"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/git"
	"github.com/EasterCompany/dex-cli/ui"
	"github.com/EasterCompany/dex-cli/utils"
)

// Update manages the unified update process for dex-cli and all other services.
// It fetches, builds, and installs all services, one by one.
func Update(args []string, buildYear string) error {
	logFile, err := config.LogFile()
	if err != nil {
		return fmt.Errorf("failed to get log file: %w", err)
	}
	defer func() { _ = logFile.Close() }()

	log := func(message string) {
		_, _ = fmt.Fprintln(logFile, message)
	}

	// Pull default Ollama models (non-fatal if it fails)
	ui.PrintHeader("Syncing Default Ollama Models")
	if err := utils.PullHardcodedModels(); err != nil {
		log(fmt.Sprintf("Failed to pull ollama models (non-fatal): %v", err))
		ui.PrintWarning("Failed to sync Ollama models. This is non-fatal and can be done manually with 'dex ollama pull'.")
	}

	log("Updating to latest version...")
	ui.PrintHeader("Updating Dexter Source Repositories")

	// ---
	// 1. Get initial versions and sizes
	// ---
	allServices := config.GetAllServices()
	oldVersions := make(map[string]string)
	oldSizes := make(map[string]int64)
	for _, s := range allServices {
		if s.IsBuildable() {
			oldVersions[s.ID] = utils.GetServiceVersion(s)
			oldSizes[s.ID] = utils.GetBinarySize(s)
		}
	}

	var dexCliDef config.ServiceDefinition
	for _, s := range allServices {
		if s.ShortName == "cli" {
			dexCliDef = s
			break
		}
	}

	// ---
	// 2. Process dex-cli FIRST (always)
	// ---
	ui.PrintInfo(ui.Colorize(fmt.Sprintf("# Updating %s", dexCliDef.ShortName), ui.ColorCyan))
	if err := utils.GitUpdateService(dexCliDef); err != nil {
		return fmt.Errorf("failed to update dex-cli source: %w", err)
	}
	if _, err := utils.RunUnifiedBuildPipeline(dexCliDef, log); err != nil {
		return err
	}
	ui.PrintSuccess(fmt.Sprintf("Successfully updated and installed %s!", dexCliDef.ShortName))
	ui.PrintInfo(fmt.Sprintf("%s  Previous Version: %s%s", ui.ColorDarkGray, oldVersions[dexCliDef.ID], ui.ColorReset))
	ui.PrintInfo(fmt.Sprintf("%s  Current Version:  %s%s", ui.ColorDarkGray, utils.GetFullServiceVersion(dexCliDef), ui.ColorReset))

	// ---
	// 3. Process all OTHER installed services
	// ---
	serviceMap, err := config.LoadServiceMapConfig()
	if err != nil {
		return fmt.Errorf("failed to load service-map.json: %w", err)
	}

	installedServices := make(map[string]bool)
	for _, serviceList := range serviceMap.Services {
		for _, serviceEntry := range serviceList {
			installedServices[serviceEntry.ID] = true
		}
	}

	for _, def := range allServices {
		if !def.IsManageable() || def.ShortName == "cli" {
			continue
		}
		if _, isInstalled := installedServices[def.ID]; !isInstalled {
			log(fmt.Sprintf("Skipping %s (not in service-map.json)", def.ShortName))
			continue
		}

		ui.PrintInfo(ui.Colorize(fmt.Sprintf("# Updating %s", def.ShortName), ui.ColorCyan))

		if err := utils.GitUpdateService(def); err != nil {
			return err
		}
		if _, err := utils.RunUnifiedBuildPipeline(def, log); err != nil {
			return err
		}
		if err := utils.InstallSystemdService(def); err != nil {
			return err
		}

		ui.PrintSuccess(fmt.Sprintf("Successfully updated and installed %s!", def.ShortName))
		parsedOldVersion := utils.ParseServiceVersionFromJSON(oldVersions[def.ID])
		ui.PrintInfo(fmt.Sprintf("%s  Previous Version: %s%s", ui.ColorDarkGray, parsedOldVersion, ui.ColorReset))
		ui.PrintInfo(fmt.Sprintf("%s  Current Version:  %s%s", ui.ColorDarkGray, utils.ParseServiceVersionFromJSON(utils.GetFullServiceVersion(def)), ui.ColorReset))
	}

	// ---
	// 4. Final Summary
	// ---
	log("Update complete.")
	ui.PrintHeader("Complete")
	time.Sleep(2 * time.Second)

	var summaryData []utils.SummaryInfo
	configuredServices, err := utils.GetConfiguredServices()
	if err != nil {
		ui.PrintWarning(fmt.Sprintf("Could not load configured services for final summary: %v", err))
	} else {
		for _, s := range configuredServices {
			if s.IsBuildable() {
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
					if oldVer.Commit == newVer.Commit {
						changeLog = "N/A"
					} else {
						repoPath, _ := config.ExpandPath(s.Source)
						changeLog, _ = git.GetCommitLogBetween(repoPath, oldVer.Commit, newVer.Commit)
					}
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
	}

	utils.PrintSummaryTable(summaryData)
	fmt.Println()
	ui.PrintSuccess("All services are up to date.")

	return nil
}
