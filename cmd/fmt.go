package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/ui"
)

// Fmt handles the 'fmt' command to format source code using Prettier and go fmt
func Fmt(args []string) error {
	ui.PrintHeader("Formatting Project Source Code")

	dexterRoot, err := config.GetDexterPath()
	if err != nil {
		return fmt.Errorf("failed to get dexter path: %w", err)
	}
	// The project root is one level up from ~/Dexter (usually ~/EasterCompany)
	projectRoot := filepath.Dir(dexterRoot)
	if _, err := os.Stat(filepath.Join(projectRoot, ".prettierrc")); err != nil {
		// Fallback to current working directory if not in standard layout
		projectRoot, _ = os.Getwd()
	}

	// 1. Run Prettier for Web Files (JS, CSS, HTML, MD, etc.)
	ui.PrintInfo("Running Prettier...")
	prettierCmd := exec.Command("prettier", "--write", ".")
	prettierCmd.Dir = projectRoot
	prettierCmd.Stdout = os.Stdout
	prettierCmd.Stderr = os.Stderr

	if err := prettierCmd.Run(); err != nil {
		ui.PrintWarning(fmt.Sprintf("Prettier encountered issues: %v", err))
	} else {
		ui.PrintSuccess("Web files formatted successfully.")
	}

	// 2. Run go fmt for Go Files
	ui.PrintInfo("Running go fmt...")
	goFmtCmd := exec.Command("go", "fmt", "./...")
	goFmtCmd.Dir = projectRoot
	goFmtCmd.Stdout = os.Stdout
	goFmtCmd.Stderr = os.Stderr

	if err := goFmtCmd.Run(); err != nil {
		ui.PrintWarning(fmt.Sprintf("go fmt encountered issues: %v", err))
	} else {
		ui.PrintSuccess("Go files formatted successfully.")
	}

	ui.PrintSuccess("Project formatting complete!")
	return nil
}
