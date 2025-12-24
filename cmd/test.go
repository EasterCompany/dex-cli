package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/ui"
	"github.com/EasterCompany/dex-cli/utils"
)

// TestResult holds the outcome of a single test operation
type TestResult struct {
	Status      string  `json:"status"` // "OK", "FAILED", "SKIPPED", "N/A"
	Message     string  `json:"message"`
	Duration    string  `json:"duration"`
	TestCount   int     `json:"test_count"`
	FailCount   int     `json:"fail_count"`
	Coverage    float64 `json:"coverage"`
	IssueCount  int     `json:"issue_count"`
	DetailsLine string  `json:"details"`
}

// ServiceTestSummary holds all test results for a single service
type ServiceTestSummary struct {
	ServiceName   string     `json:"service_name"`
	FormatResult  TestResult `json:"format_result"`
	LintResult    TestResult `json:"lint_result"`
	TestResult    TestResult `json:"test_result"`
	TotalDuration string     `json:"total_duration"`
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

	// Determine which services to test
	var servicesToTest []config.ServiceDefinition
	allServices := config.GetAllServices()

	if serviceName != "" {
		// Test a specific service
		found := false
		for _, s := range allServices {
			if s.ShortName == serviceName || s.ID == serviceName {
				servicesToTest = append(servicesToTest, s)
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("service '%s' not found", serviceName)
		}
	} else {
		// Test all buildable services
		for _, s := range allServices {
			if s.IsBuildable() {
				sourcePath, err := config.ExpandPath(s.Source)
				if err == nil {
					if _, err := os.Stat(sourcePath); err == nil {
						servicesToTest = append(servicesToTest, s)
					}
				}
			}
		}
	}

	if len(servicesToTest) == 0 {
		return fmt.Errorf("no services found to test")
	}

	// Print header
	ui.PrintHeader("Testing All Services")

	var summaries []ServiceTestSummary

	// Test each service
	for _, def := range servicesToTest {
		fmt.Println()
		ui.PrintInfo(fmt.Sprintf("%s%s%s", ui.ColorCyan, fmt.Sprintf("Testing %s", def.ShortName), ui.ColorReset))

		sourcePath, err := config.ExpandPath(def.Source)
		if err != nil {
			ui.PrintError(fmt.Sprintf("Failed to expand source path: %v", err))
			continue
		}

		if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
			ui.PrintWarning(fmt.Sprintf("Skipping %s: source code not found at %s. Run 'dex add' to download & install it.", def.ShortName, sourcePath))
			continue
		}

		startTime := time.Now()

		// Run format, lint, test
		formatResult := runFormatCheck(def, sourcePath, log)
		lintResult := runLintCheck(def, sourcePath, log)
		testResult := runTestCheck(def, sourcePath, log)

		totalDuration := time.Since(startTime)

		// Print individual results
		printTestStepResult("Format", formatResult)
		printTestStepResult("Lint", lintResult)
		printTestStepResult("Test", testResult)

		// Print duration
		ui.PrintInfo(fmt.Sprintf("%s  Total Duration: %s%s", ui.ColorDarkGray, totalDuration.Round(time.Millisecond), ui.ColorReset))

		summary := ServiceTestSummary{
			ServiceName:   def.ShortName,
			FormatResult:  formatResult,
			LintResult:    lintResult,
			TestResult:    testResult,
			TotalDuration: totalDuration.String(),
		}
		summaries = append(summaries, summary)

		// EMIT EVENT: system.test.completed
		utils.SendEvent("system.test.completed", map[string]interface{}{
			"service_name": def.ShortName,
			"format":       summary.FormatResult,
			"lint":         summary.LintResult,
			"test":         summary.TestResult,
			"duration":     summary.TotalDuration,
		})

		// EMIT NOTIFICATION IF FAILED
		if summary.FormatResult.Status == "FAILED" || summary.LintResult.Status == "FAILED" || summary.TestResult.Status == "FAILED" {
			failedSteps := []string{}
			if summary.FormatResult.Status == "FAILED" {
				failedSteps = append(failedSteps, "Formatting")
			}
			if summary.LintResult.Status == "FAILED" {
				failedSteps = append(failedSteps, fmt.Sprintf("Linting (%d issues)", summary.LintResult.IssueCount))
			}
			if summary.TestResult.Status == "FAILED" {
				failedSteps = append(failedSteps, fmt.Sprintf("Tests (%d/%d failed)", summary.TestResult.FailCount, summary.TestResult.TestCount))
			}

			utils.SendEvent("system.notification.generated", map[string]interface{}{
				"title":    fmt.Sprintf("Test Failure: %s", def.ShortName),
				"priority": "high",
				"category": "error",
				"body":     fmt.Sprintf("The service '%s' failed the following test steps: %s. Please review the logs or run 'dex test %s' for details.", def.ShortName, strings.Join(failedSteps, ", "), def.ShortName),
			})
		}
	}

	// Print summary
	fmt.Println()
	ui.PrintHeader("Test Summary")

	printTestSummaryTable(summaries)

	// Determine overall status
	allPassed := true
	for _, s := range summaries {
		if s.FormatResult.Status == "FAILED" || s.LintResult.Status == "FAILED" || s.TestResult.Status == "FAILED" {
			allPassed = false
			break
		}
	}

	fmt.Println()
	if allPassed {
		ui.PrintSuccess("All tests passed!")
		// Optional: Emit a positive summary notification if multiple services were tested
		if len(summaries) > 1 {
			utils.SendEvent("system.notification.generated", map[string]interface{}{
				"title":    "All Systems Verified",
				"priority": "low",
				"category": "system",
				"body":     fmt.Sprintf("Successfully ran tests for %d services. All checks (Format, Lint, Test) passed.", len(summaries)),
			})
		}
	} else {
		ui.PrintError("Some tests failed.")
	}

	return nil
}

// runFormatCheck runs formatting checks for a service
func runFormatCheck(def config.ServiceDefinition, sourcePath string, log func(string)) TestResult {
	startTime := time.Now()

	ui.PrintInfo("Checking formatting...")

	// For Go projects, run gofmt in check mode
	if def.Type == "cli" || strings.HasPrefix(def.ID, "dex-") {
		cmd := exec.Command("gofmt", "-l", ".")
		cmd.Dir = sourcePath
		output, err := cmd.CombinedOutput()
		duration := time.Since(startTime)

		if err != nil {
			log(fmt.Sprintf("[%s] Format check failed: %v", def.ShortName, err))
			return TestResult{
				Status:   "FAILED",
				Message:  fmt.Sprintf("gofmt failed: %v", err),
				Duration: duration.String(),
			}
		}

		// Check if any files need formatting
		outputStr := strings.TrimSpace(string(output))
		if outputStr != "" {
			allFiles := strings.Split(outputStr, "\n")
			var files []string
			for _, f := range allFiles {
				if !strings.Contains(f, "vendor/") {
					files = append(files, f)
				}
			}

			fileCount := len(files)
			if fileCount > 0 {
				log(fmt.Sprintf("[%s] Format check found %d files needing formatting", def.ShortName, fileCount))
				return TestResult{
					Status:      "FAILED",
					Message:     fmt.Sprintf("%d file(s) need formatting: %s", fileCount, strings.Join(files, ", ")),
					Duration:    duration.String(),
					IssueCount:  fileCount,
					DetailsLine: fmt.Sprintf("%d file(s) unformatted", fileCount),
				}
			}
		}

		return TestResult{
			Status:      "OK",
			Duration:    duration.String(),
			DetailsLine: "all files formatted",
		}
	}

	return TestResult{Status: "SKIPPED", Message: "formatting not configured for this service type", Duration: time.Since(startTime).String()}
}

// runLintCheck runs linting checks for a service
func runLintCheck(def config.ServiceDefinition, sourcePath string, log func(string)) TestResult {
	startTime := time.Now()

	ui.PrintInfo("Linting...")

	// For Go projects, run golangci-lint
	if def.Type == "cli" || strings.HasPrefix(def.ID, "dex-") {
		// Check if golangci-lint is available
		if _, err := exec.LookPath("golangci-lint"); err != nil {
			return TestResult{Status: "SKIPPED", Message: "golangci-lint not found", Duration: time.Since(startTime).String()}
		}

		cmd := exec.Command("golangci-lint", "run")
		cmd.Dir = sourcePath
		output, err := cmd.CombinedOutput()
		duration := time.Since(startTime)

		outputStr := strings.TrimSpace(string(output))

		if err != nil {
			// Parse the number of issues
			issueCount := countLintIssues(outputStr)
			log(fmt.Sprintf("[%s] Lint found %d issues", def.ShortName, issueCount))

			// Truncate output if too long
			displayOutput := outputStr
			if len(displayOutput) > 500 {
				displayOutput = displayOutput[:500] + "... (truncated)"
			}

			return TestResult{
				Status:      "FAILED",
				Message:     displayOutput,
				Duration:    duration.String(),
				IssueCount:  issueCount,
				DetailsLine: fmt.Sprintf("%d issue(s)", issueCount),
			}
		}

		// Check if there were issues even without error
		issueCount := countLintIssues(outputStr)
		if issueCount > 0 {
			log(fmt.Sprintf("[%s] Lint found %d issues (non-fatal)", def.ShortName, issueCount))
			return TestResult{
				Status:      "FAILED",
				Message:     outputStr,
				Duration:    duration.String(),
				IssueCount:  issueCount,
				DetailsLine: fmt.Sprintf("%d issue(s)", issueCount),
			}
		}

		return TestResult{
			Status:      "OK",
			Duration:    duration.String(),
			DetailsLine: "no issues",
		}
	}

	return TestResult{Status: "SKIPPED", Message: "linting not configured for this service type", Duration: time.Since(startTime).String()}
}

// runTestCheck runs unit tests for a service
func runTestCheck(def config.ServiceDefinition, sourcePath string, log func(string)) TestResult {
	startTime := time.Now()

	ui.PrintInfo("Running tests...")

	// For Go projects, run go test
	if def.Type == "cli" || strings.HasPrefix(def.ID, "dex-") {
		cmd := exec.Command("go", "test", "-v", "-cover", "./...")
		cmd.Dir = sourcePath
		output, err := cmd.CombinedOutput()
		duration := time.Since(startTime)

		outputStr := string(output)

		// Parse test results
		testCount, failCount, coverage := parseGoTestOutput(outputStr)

		if err != nil {
			log(fmt.Sprintf("[%s] Tests failed: %d/%d failed", def.ShortName, failCount, testCount))

			// Extract failure details
			failureDetails := extractTestFailures(outputStr)

			return TestResult{
				Status:      "FAILED",
				Message:     failureDetails,
				Duration:    duration.String(),
				TestCount:   testCount,
				FailCount:   failCount,
				Coverage:    coverage,
				DetailsLine: fmt.Sprintf("%d/%d passed, %.1f%% coverage", testCount-failCount, testCount, coverage),
			}
		}

		// Check if there are no test files
		if strings.Contains(outputStr, "[no test files]") {
			return TestResult{
				Status:      "SKIPPED",
				Message:     "no test files found",
				Duration:    duration.String(),
				DetailsLine: "no test files",
			}
		}

		detailsLine := fmt.Sprintf("%d passed", testCount)
		if coverage > 0 {
			detailsLine += fmt.Sprintf(", %.1f%% coverage", coverage)
		}

		return TestResult{
			Status:      "OK",
			Duration:    duration.String(),
			TestCount:   testCount,
			Coverage:    coverage,
			DetailsLine: detailsLine,
		}
	}

	return TestResult{Status: "SKIPPED", Message: "testing not configured for this service type", Duration: time.Since(startTime).String()}
}

// printTestStepResult prints the result of a single test step
func printTestStepResult(stepName string, result TestResult) {
	icon := ""
	color := ui.ColorWhite

	switch result.Status {
	case "OK":
		icon = "✓"
		color = ui.ColorGreen
	case "FAILED":
		icon = "✕"
		color = ui.ColorRed
	case "SKIPPED":
		icon = "○"
		color = ui.ColorYellow
	case "N/A":
		icon = "−"
		color = ui.ColorDarkGray
	}

	// Build the status line
	statusLine := fmt.Sprintf("%s%s %s", color, icon, stepName)
	if result.DetailsLine != "" {
		statusLine += fmt.Sprintf(" (%s)", result.DetailsLine)
	}
	if result.Duration != "" {
		parsed, _ := time.ParseDuration(result.Duration)
		statusLine += fmt.Sprintf(" [%s]", parsed.Round(time.Millisecond))
	}
	statusLine += ui.ColorReset

	fmt.Println(statusLine)

	// Print detailed message if failed
	if result.Status == "FAILED" && result.Message != "" {
		// Indent and print message
		lines := strings.Split(result.Message, "\n")
		for i, line := range lines {
			if i < 10 { // Limit to first 10 lines
				ui.PrintInfo(fmt.Sprintf("  %s", line))
			}
		}
		if len(lines) > 10 {
			ui.PrintInfo(fmt.Sprintf("  ... and %d more lines (see log for details)", len(lines)-10))
		}
	}
}

// printTestSummaryTable prints the final summary table
func printTestSummaryTable(summaries []ServiceTestSummary) {
	// Define fixed/max column widths for consistent formatting
	// Service: 15, Format: 12, Lint: 12, Test: 12, Duration: 10
	table := ui.NewTableWithWidths(
		[]string{"SERVICE", "FORMAT", "LINT", "TEST", "DURATION"},
		[]int{15, 12, 12, 12, 10},
	)

	for _, s := range summaries {
		formatStatus := formatStatusForTable(s.FormatResult)
		lintStatus := formatStatusForTable(s.LintResult)
		testStatus := formatStatusForTable(s.TestResult)
		duration := s.TotalDuration

		table.AddRow(ui.TableRow{
			s.ServiceName,
			formatStatus,
			lintStatus,
			testStatus,
			duration,
		})
	}

	table.Render()
}

// formatStatusForTable formats a test result for table display
func formatStatusForTable(result TestResult) string {
	switch result.Status {
	case "OK":
		return ui.Colorize("✓ PASS", ui.ColorGreen)
	case "FAILED":
		if result.IssueCount > 0 {
			return ui.Colorize(fmt.Sprintf("✕ %d issue(s)", result.IssueCount), ui.ColorRed)
		}
		if result.FailCount > 0 {
			return ui.Colorize(fmt.Sprintf("✕ %d/%d failed", result.FailCount, result.TestCount), ui.ColorRed)
		}
		return ui.Colorize("✕ FAIL", ui.ColorRed)
	case "SKIPPED":
		return ui.Colorize("○ SKIP", ui.ColorYellow)
	case "N/A":
		return ui.Colorize("− N/A", ui.ColorDarkGray)
	default:
		return ui.Colorize("?", ui.ColorDarkGray)
	}
}

// countLintIssues counts the number of linting issues in the output
func countLintIssues(output string) int {
	if output == "" {
		return 0
	}
	// Count lines that look like lint issues (typically have a file path and line number)
	lines := strings.Split(output, "\n")
	count := 0
	for _, line := range lines {
		// Look for patterns like "file.go:123:45: issue"
		if strings.Contains(line, ".go:") && strings.Contains(line, ":") {
			count++
		}
	}
	return count
}

// parseGoTestOutput parses go test output to extract test count, failures, and coverage
func parseGoTestOutput(output string) (testCount, failCount int, coverage float64) {
	lines := strings.Split(output, "\n")

	// Count tests and failures
	for _, line := range lines {
		// Look for test result lines like "--- PASS: TestName" or "--- FAIL: TestName"
		if strings.HasPrefix(line, "--- PASS:") {
			testCount++
		} else if strings.HasPrefix(line, "--- FAIL:") {
			testCount++
			failCount++
		}

		// Look for coverage line like "coverage: 75.5% of statements"
		if strings.Contains(line, "coverage:") && strings.Contains(line, "% of statements") {
			re := regexp.MustCompile(`coverage:\s+([\d.]+)%`)
			matches := re.FindStringSubmatch(line)
			if len(matches) > 1 {
				if cov, err := strconv.ParseFloat(matches[1], 64); err == nil {
					coverage = cov
				}
			}
		}
	}

	return
}

// extractTestFailures extracts the most relevant failure information from test output
func extractTestFailures(output string) string {
	lines := strings.Split(output, "\n")
	var failures []string

	for _, line := range lines {
		// Look for FAIL lines
		if strings.Contains(line, "FAIL") || strings.Contains(line, "Error:") || strings.Contains(line, "panic:") {
			failures = append(failures, strings.TrimSpace(line))
		}
	}

	if len(failures) == 0 {
		return "Tests failed (see log for details)"
	}

	// Return first few failures
	if len(failures) > 5 {
		return strings.Join(failures[:5], "\n") + "\n... (more failures in log)"
	}

	return strings.Join(failures, "\n")
}
