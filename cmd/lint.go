package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/ui"
)

// Lint handles the 'lint' command to run linters on source code
func Lint(args []string) error {
	ui.PrintHeader("Linting Project Source Code")

	dexterRoot, err := config.GetDexterPath()
	if err != nil {
		return fmt.Errorf("failed to get dexter path: %w", err)
	}
	// The project root is one level up from ~/Dexter (usually ~/EasterCompany)
	projectRoot := filepath.Dir(dexterRoot)
	if _, err := os.Stat(filepath.Join(projectRoot, ".eslintrc.json")); err != nil {
		// Fallback to current working directory if not in standard layout
		projectRoot, _ = os.Getwd()
	}

	hasErrors := false

	// 1. Run ESLint for JS
	ui.PrintInfo("Running ESLint (JS)...")
	if _, err := exec.LookPath("eslint"); err == nil {
		// Target specific directories or file types to avoid scanning node_modules or go files
		// We scan the current directory but eslint ignores .gitignore/node_modules usually
		// Better to target "." and let config handle ignores
		eslintCmd := exec.Command("eslint", ".", "--ext", ".js,.html")
		eslintCmd.Dir = projectRoot
		eslintCmd.Stdout = os.Stdout
		eslintCmd.Stderr = os.Stderr

		if err := eslintCmd.Run(); err != nil {
			ui.PrintError(fmt.Sprintf("ESLint found issues: %v", err))
			hasErrors = true
		} else {
			ui.PrintSuccess("ESLint passed.")
		}
	} else {
		ui.PrintWarning("eslint not found, skipping JS linting.")
	}

	// 2. Run Stylelint for CSS
	ui.PrintInfo("Running Stylelint (CSS)...")
	if _, err := exec.LookPath("stylelint"); err == nil {
		stylelintCmd := exec.Command("stylelint", "**/*.css")
		stylelintCmd.Dir = projectRoot
		stylelintCmd.Stdout = os.Stdout
		stylelintCmd.Stderr = os.Stderr

		if err := stylelintCmd.Run(); err != nil {
			ui.PrintError(fmt.Sprintf("Stylelint found issues: %v", err))
			hasErrors = true
		} else {
			ui.PrintSuccess("Stylelint passed.")
		}
	} else {
		ui.PrintWarning("stylelint not found, skipping CSS linting.")
	}

	// 3. Run HTMLHint for HTML
	ui.PrintInfo("Running HTMLHint (HTML)...")
	if _, err := exec.LookPath("htmlhint"); err == nil {
		htmlhintCmd := exec.Command("htmlhint", "**/*.html")
		htmlhintCmd.Dir = projectRoot
		htmlhintCmd.Stdout = os.Stdout
		htmlhintCmd.Stderr = os.Stderr

		if err := htmlhintCmd.Run(); err != nil {
			ui.PrintError(fmt.Sprintf("HTMLHint found issues: %v", err))
			hasErrors = true
		} else {
			ui.PrintSuccess("HTMLHint passed.")
		}
	} else {
		ui.PrintWarning("htmlhint not found, skipping HTML linting.")
	}

	// 4. Run golangci-lint for Go
	ui.PrintInfo("Running golangci-lint (Go)...")
	if _, err := exec.LookPath("golangci-lint"); err == nil {
		// golangci-lint typically runs per module. Running from root might fail if root isn't a module.
		// We should probably run it on specific service directories or recursively?
		// Recursive run might be complex. For now, let's run on standard paths if they exist.
		dirs := []string{"dex-cli", "dex-event-service", "dex-discord-service", "dex-web-service"}

		for _, dir := range dirs {
			fullPath := filepath.Join(projectRoot, dir)
			if _, err := os.Stat(fullPath); err == nil {
				ui.PrintInfo(fmt.Sprintf("  Linting %s...", dir))
				goLintCmd := exec.Command("golangci-lint", "run")
				goLintCmd.Dir = fullPath
				goLintCmd.Stdout = os.Stdout
				goLintCmd.Stderr = os.Stderr
				if err := goLintCmd.Run(); err != nil {
					ui.PrintError(fmt.Sprintf("golangci-lint found issues in %s: %v", dir, err))
					hasErrors = true
				}
			}
		}
	} else {
		ui.PrintWarning("golangci-lint not found, skipping Go linting.")
	}

	if hasErrors {
		return fmt.Errorf("linting failed with errors")
	}

	ui.PrintSuccess("All linting checks passed!")
	return nil
}
