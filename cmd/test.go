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

// Test runs format, lint, and test for all services
func Test(args []string) error {
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

	projects, err := filepath.Glob(filepath.Join(easterCompanyPath, "*"))
	if err != nil {
		return err
	}

	var overallResults []ui.TableRow

	for _, project := range projects {
		projectName := filepath.Base(project)
		if !strings.HasPrefix(projectName, "dex-") && projectName != "easter.company" {
			continue
		}

		fmt.Printf("▼ %s\n", projectName)
		log(fmt.Sprintf("Testing project: %s", projectName))

		// Format
		formatResult := formatProject(project, log)
		fmt.Printf("  Formatting: %s\n", formatResult)
		log(fmt.Sprintf("  Formatting result: %s", formatResult))

		// Lint
		lintResult := lintProject(project, log)
		fmt.Printf("  Linting: %s\n", lintResult)
		log(fmt.Sprintf("  Linting result: %s", lintResult))

		// Test
		testResult := "SKIPPED"
		if strings.HasPrefix(projectName, "dex-") {
			testResult = testProject(project, log)
			fmt.Printf("  Testing: %s\n", testResult)
			log(fmt.Sprintf("  Testing result: %s", testResult))
		}

		overallResults = append(overallResults, ui.TableRow{projectName, formatResult, lintResult, testResult})
	}

	fmt.Println("\n--- Overall Results ---")
	table := ui.NewTable([]string{"Project", "Format", "Lint", "Test"})
	for _, row := range overallResults {
		table.AddRow(row)
	}
	table.Render()

	return nil
}

func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

func formatProject(projectPath string, log func(string)) string {
	// Go
	if commandExists("gofmt") {
		cmd := exec.Command("gofmt", "-w", ".")
		cmd.Dir = projectPath
		if err := cmd.Run(); err != nil {
			return "Go: FAILED"
		}
	}

	// Prettier for HTML, CSS, JS, TS, JSON, Markdown
	if commandExists("prettier") {
		cmd := exec.Command("prettier", "--write", "--prose-wrap", "always", ".")
		cmd.Dir = projectPath
		_ = cmd.Run() // Prettier returns error if no files found, so we ignore it
	}

	// Shell
	if commandExists("shfmt") {
		cmd := exec.Command("shfmt", "-w", ".")
		cmd.Dir = projectPath
		if err := cmd.Run(); err != nil {
			return "Shell: FAILED"
		}
	}

	// Python
	if commandExists("black") {
		cmd := exec.Command("black", ".")
		cmd.Dir = projectPath
		if err := cmd.Run(); err != nil {
			return "Python: FAILED"
		}
	}

	return "PASSED"
}

func lintProject(projectPath string, log func(string)) string {
	// Go
	if commandExists("golangci-lint") {
		cmd := exec.Command("golangci-lint", "run")
		cmd.Dir = projectPath
		if err := cmd.Run(); err != nil {
			return "Go: FAILED"
		}
	}

	// ESLint for JS/TS
	if commandExists("eslint") {
		cmd := exec.Command("eslint", ".")
		cmd.Dir = projectPath
		if err := cmd.Run(); err != nil {
			return "JS/TS: FAILED"
		}
	}

	// Stylelint for CSS
	if commandExists("stylelint") {
		cmd := exec.Command("stylelint", "**/*.css")
		cmd.Dir = projectPath
		if err := cmd.Run(); err != nil {
			return "CSS: FAILED"
		}
	}

	// HTMLHint for HTML
	if commandExists("htmlhint") {
		cmd := exec.Command("htmlhint", "**/*.html")
		cmd.Dir = projectPath
		if err := cmd.Run(); err != nil {
			return "HTML: FAILED"
		}
	}

	// JSONLint for JSON
	if commandExists("jsonlint") {
		cmd := exec.Command("jsonlint", "-q", ".")
		cmd.Dir = projectPath
		if err := cmd.Run(); err != nil {
			return "JSON: FAILED"
		}
	}

	// Markdown
	if commandExists("markdownlint") {
		cmd := exec.Command("markdownlint", ".")
		cmd.Dir = projectPath
		if err := cmd.Run(); err != nil {
			return "Markdown: FAILED"
		}
	}

	// Shell
	if commandExists("shellcheck") {
		cmd := exec.Command("shellcheck", ".")
		cmd.Dir = projectPath
		if err := cmd.Run(); err != nil {
			return "Shell: FAILED"
		}
	}

	// Python
	if commandExists("flake8") {
		cmd := exec.Command("flake8", ".")
		cmd.Dir = projectPath
		if err := cmd.Run(); err != nil {
			return "Python: FAILED"
		}
	}

	return "PASSED"
}

func testProject(projectPath string, log func(string)) string {
	// Go
	if _, err := os.Stat(filepath.Join(projectPath, "go.mod")); err == nil {
		cmd := exec.Command("go", "test", "./...")
		cmd.Dir = projectPath
		if err := cmd.Run(); err != nil {
			return "Go: FAILED"
		}
	}

	// Python
	if _, err := os.Stat(filepath.Join(projectPath, "requirements.txt")); err == nil {
		if commandExists("pytest") {
			cmd := exec.Command("pytest")
			cmd.Dir = projectPath
			if err := cmd.Run(); err != nil {
				return "Python: FAILED"
			}
		}
	}

	return "PASSED"
}
