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

func getExecutablePath(name string) (string, bool) {
	virtualEnvPath, err := config.ExpandPath("~/Dexter/python/bin/" + name)
	if err == nil {
		if _, err := os.Stat(virtualEnvPath); err == nil {
			return virtualEnvPath, true
		}
	}

	// Fallback to system path
	path, err := exec.LookPath(name)
	if err != nil {
		return "", false
	}
	return path, true
}

func formatProject(projectPath string, log func(string)) string {
	// Go
	if gofmtPath, ok := getExecutablePath("gofmt"); ok {
		cmd := exec.Command(gofmtPath, "-w", ".")
		cmd.Dir = projectPath
		if err := cmd.Run(); err != nil {
			return "Go: FAILED"
		}
	}

	// Prettier for HTML, CSS, JS, TS, JSON, Markdown
	if prettierPath, ok := getExecutablePath("prettier"); ok {
		cmd := exec.Command(prettierPath, "--write", "--prose-wrap", "always", ".")
		cmd.Dir = projectPath
		_ = cmd.Run() // Prettier returns error if no files found, so we ignore it
	}

	// Shell
	if shfmtPath, ok := getExecutablePath("shfmt"); ok {
		cmd := exec.Command(shfmtPath, "-w", ".")
		cmd.Dir = projectPath
		if err := cmd.Run(); err != nil {
			return "Shell: FAILED"
		}
	}

	// Python
	if blackPath, ok := getExecutablePath("black"); ok {
		cmd := exec.Command(blackPath, ".")
		cmd.Dir = projectPath
		if err := cmd.Run(); err != nil {
			return "Python: FAILED"
		}
	}

	return "PASSED"
}

func lintProject(projectPath string, log func(string)) string {
	// Go
	if golintPath, ok := getExecutablePath("golangci-lint"); ok {
		cmd := exec.Command(golintPath, "run")
		cmd.Dir = projectPath
		if err := cmd.Run(); err != nil {
			return "Go: FAILED"
		}
	}

	// ESLint for JS/TS
	if eslintPath, ok := getExecutablePath("eslint"); ok {
		cmd := exec.Command(eslintPath, ".")
		cmd.Dir = projectPath
		if err := cmd.Run(); err != nil {
			return "JS/TS: FAILED"
		}
	}

	// Stylelint for CSS
	if stylelintPath, ok := getExecutablePath("stylelint"); ok {
		cmd := exec.Command(stylelintPath, "**/*.css")
		cmd.Dir = projectPath
		if err := cmd.Run(); err != nil {
			return "CSS: FAILED"
		}
	}

	// HTMLHint for HTML
	if htmlhintPath, ok := getExecutablePath("htmlhint"); ok {
		cmd := exec.Command(htmlhintPath, "**/*.html")
		cmd.Dir = projectPath
		if err := cmd.Run(); err != nil {
			return "HTML: FAILED"
		}
	}

	// JSONLint for JSON
	if jsonlintPath, ok := getExecutablePath("jsonlint"); ok {
		cmd := exec.Command(jsonlintPath, "-q", ".")
		cmd.Dir = projectPath
		if err := cmd.Run(); err != nil {
			return "JSON: FAILED"
		}
	}

	// Markdown
	if markdownlintPath, ok := getExecutablePath("markdownlint"); ok {
		cmd := exec.Command(markdownlintPath, ".")
		cmd.Dir = projectPath
		if err := cmd.Run(); err != nil {
			return "Markdown: FAILED"
		}
	}

	// Shell
	if shellcheckPath, ok := getExecutablePath("shellcheck"); ok {
		cmd := exec.Command(shellcheckPath, ".")
		cmd.Dir = projectPath
		if err := cmd.Run(); err != nil {
			return "Shell: FAILED"
		}
	}

	// Python
	if flake8Path, ok := getExecutablePath("flake8"); ok {
		cmd := exec.Command(flake8Path, ".")
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
		if goPath, ok := getExecutablePath("go"); ok {
			cmd := exec.Command(goPath, "test", "./...")
			cmd.Dir = projectPath
			if err := cmd.Run(); err != nil {
				return "Go: FAILED"
			}
		}
	}

	// Python
	if _, err := os.Stat(filepath.Join(projectPath, "requirements.txt")); err == nil {
		if pytestPath, ok := getExecutablePath("pytest"); ok {
			cmd := exec.Command(pytestPath)
			cmd.Dir = projectPath
			if err := cmd.Run(); err != nil {
				return "Python: FAILED"
			}
		}
	}

	return "PASSED"
}
