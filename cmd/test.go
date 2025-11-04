package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/ui"
)

type Result struct {
	Status  string
	Message string
	Tool    string
}

// Test runs format, lint, and test for all services, or a specific service if provided.
func Test(args []string) error {
	var serviceName string
	if len(args) > 0 {
		serviceName = args[0]
	}

	logFile, err := config.LogFile()
	if err != nil {
		return fmt.Errorf("failed to get log file: %w", err)
	}
	defer func() { _ = logFile.Close() }()

	log := func(message string) {
		_, _ = fmt.Fprintln(logFile, message)
	}

	log("Running test command...")

	easterCompanyPath, err := config.ExpandPath("~/EasterCompany")
	if err != nil {
		return err
	}

	var projects []string
	if serviceName != "" {
		// If a service name is provided, test only that service
		resolvedServiceName, err := resolveServiceName(serviceName)
		if err != nil {
			return err
		}
		projectPath := filepath.Join(easterCompanyPath, resolvedServiceName)
		if _, err := os.Stat(projectPath); os.IsNotExist(err) {
			return fmt.Errorf("service '%s' not found in %s", resolvedServiceName, easterCompanyPath)
		}
		projects = []string{projectPath}
	} else {
		// Otherwise, test all applicable services
		allProjects, err := filepath.Glob(filepath.Join(easterCompanyPath, "*"))
		if err != nil {
			return err
		}
		for _, project := range allProjects {
			projectName := filepath.Base(project)
			if strings.HasPrefix(projectName, "dex-") || projectName == "easter.company" {
				projects = append(projects, project)
			}
		}
	}

	var overallResults []ui.TableRow

	for _, project := range projects {
		projectName := filepath.Base(project)

		fmt.Printf("▼ %s\n", projectName)
		log(fmt.Sprintf("Testing project: %s", projectName))

		formatResult := formatProject(project, log)
		lintResult := lintProject(project, log)
		testResult := testProject(project, log)

		isSingleService := len(projects) == 1
		printDetailedResults("Formatting", formatResult, isSingleService)
		printDetailedResults("Linting", lintResult, isSingleService)
		printDetailedResults("Testing", testResult, isSingleService)

		overallResults = append(overallResults, ui.TableRow{projectName, formatResult.Status, lintResult.Status, testResult.Status})
	}

	if len(projects) > 1 {
		fmt.Println("\n--- Overall Results ---")
		table := ui.NewTable([]string{"Project", "Format", "Lint", "Test"})
		for _, row := range overallResults {
			table.AddRow(row)
		}
		table.Render()
	}

	return nil
}

func printDetailedResults(category string, result Result, isSingleService bool) {
	if isSingleService {
		fmt.Printf("  %s: %s (%s)\n", category, result.Status, result.Tool)
		if result.Message != "" {
			fmt.Printf("    └ %s\n", result.Message)
		}
	} else {
		switch result.Status {
		case "BAD":
			fmt.Printf("  %s: %s\n", category, result.Status)
			fmt.Printf("    └ %s\n", result.Message)
		case "OK":
			fmt.Printf("  %s: %s\n", category, result.Status)
		}
	}
}

func getExecutablePath(name string) (string, bool) {
	virtualEnvPath, err := config.ExpandPath("~/Dexter/python/bin/" + name)
	if err == nil {
		if _, err := os.Stat(virtualEnvPath); err == nil {
			return virtualEnvPath, true
		}
	}

	path, err := exec.LookPath(name)
	return path, err == nil
}

func hasFiles(projectPath, pattern string) bool {
	globPattern := filepath.Join(projectPath, pattern)
	matches, err := filepath.Glob(globPattern)
	return err == nil && len(matches) > 0
}

func hasConfigFile(projectPath, configFileName string) bool {
	_, err := os.Stat(filepath.Join(projectPath, configFileName))
	return err == nil
}

func runCheck(projectPath, toolName, filePattern string, configFiles []string, args ...string) Result {
	if !hasFiles(projectPath, filePattern) {
		return Result{Status: "N/A", Tool: toolName}
	}

	hasConfig := false
	if len(configFiles) == 0 {
		hasConfig = true // No config file needed
	} else {
		for _, configFile := range configFiles {
			if hasConfigFile(projectPath, configFile) {
				hasConfig = true
				break
			}
		}
	}

	if !hasConfig {
		return Result{Status: "N/A", Tool: toolName, Message: fmt.Sprintf("No config for %s found", toolName)}
	}

	executable, ok := getExecutablePath(toolName)
	if !ok {
		return Result{Status: "N/A", Tool: toolName, Message: fmt.Sprintf("%s not found", toolName)}
	}

	cmd := exec.Command(executable, args...)
	cmd.Dir = projectPath
	if output, err := cmd.CombinedOutput(); err != nil {
		// Ignore "no files found" errors from prettier
		if toolName == "prettier" && strings.Contains(string(output), "No files matching") {
			return Result{Status: "OK", Tool: toolName}
		}
		msg := fmt.Sprintf("%s failed: %s", toolName, string(output))
		return Result{Status: "BAD", Tool: toolName, Message: msg}
	}

	return Result{Status: "OK", Tool: toolName}
}

func formatProject(projectPath string, log func(string)) Result {
	projectName := filepath.Base(projectPath)
	var results []Result
	if strings.HasPrefix(projectName, "dex-") {
		results = []Result{
			runCheck(projectPath, "gofmt", "**/*.go", nil, "-w", "."),
		}
	} else if projectName == "easter.company" {
		results = []Result{
			runCheck(projectPath, "prettier", "**/*.js", []string{".prettierrc", ".prettierrc.json", ".prettierrc.js"}, "--write", "**/*.js"),
			runCheck(projectPath, "prettier", "**/*.ts", []string{".prettierrc", ".prettierrc.json", ".prettierrc.js"}, "--write", "**/*.ts"),
			runCheck(projectPath, "prettier", "**/*.css", []string{".prettierrc", ".prettierrc.json", ".prettierrc.js"}, "--write", "**/*.css"),
			runCheck(projectPath, "prettier", "**/*.html", []string{".prettierrc", ".prettierrc.json", ".prettierrc.js"}, "--write", "**/*.html"),
			runCheck(projectPath, "prettier", "**/*.json", []string{".prettierrc", ".prettierrc.json", ".prettierrc.js"}, "--write", "**/*.json"),
			runCheck(projectPath, "prettier", "**/*.md", []string{".prettierrc", ".prettierrc.json", ".prettierrc.js"}, "--write", "**/*.md"),
			runCheck(projectPath, "shfmt", "**/*.sh", nil, "-w", "**/*.sh"),
		}
	}
	return aggregateResults(results)
}

func lintProject(projectPath string, log func(string)) Result {
	projectName := filepath.Base(projectPath)
	var results []Result
	if strings.HasPrefix(projectName, "dex-") {
		results = []Result{
			runCheck(projectPath, "golangci-lint", "**/*.go", []string{".golangci.yml", ".golangci.yaml", ".golangci.json"}, "run"),
		}
	} else if projectName == "easter.company" {
		results = []Result{
			runCheck(projectPath, "eslint", "**/*.js", []string{".eslintrc.js", ".eslintrc.json"}, "**/*.js"),
			runCheck(projectPath, "eslint", "**/*.ts", []string{".eslintrc.js", ".eslintrc.json"}, "**/*.ts"),
			runCheck(projectPath, "stylelint", "**/*.css", []string{".stylelintrc.js", ".stylelintrc.json"}, "**/*.css"),
			runCheck(projectPath, "htmlhint", "**/*.html", []string{".htmlhintrc"}, "**/*.html"),
		}
	}
	return aggregateResults(results)
}

func testProject(projectPath string, log func(string)) Result {
	projectName := filepath.Base(projectPath)
	var results []Result
	if strings.HasPrefix(projectName, "dex-") {
		results = []Result{
			runCheck(projectPath, "go", "go.mod", nil, "test", "-v", "./..."),
		}
	}
	return aggregateResults(results)
}

func aggregateResults(results []Result) Result {
	finalResult := Result{Status: "N/A"}
	badMessages := []string{}

	hasApplicable := false
	for _, r := range results {
		if r.Status == "BAD" {
			finalResult.Status = "BAD"
			badMessages = append(badMessages, r.Message)
		}
		if r.Status != "N/A" {
			hasApplicable = true
		}
	}

	if hasApplicable && finalResult.Status != "BAD" {
		finalResult.Status = "OK"
	}

	if len(badMessages) > 0 {
		finalResult.Message = strings.Join(badMessages, "; ")
	}

	return finalResult
}

func resolveServiceName(shortName string) (string, error) {
	for _, service := range config.GetAllServices() {
		if service.ShortName == shortName {
			return service.ID, nil
		}
	}
	// Fallback to checking the full ID if no short name matches
	for _, service := range config.GetAllServices() {
		if service.ID == shortName {
			return service.ID, nil
		}
	}
	return "", fmt.Errorf("service with name '%s' not found", shortName)
}
