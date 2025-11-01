package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/EasterCompany/dex-cli/ui"
)

// Format formats and lints the code
func Format(args []string) error {
	fmt.Println(ui.RenderTitle("FORMATTING & LINTING"))

	if err := formatGoFiles(); err != nil {
		return err
	}

	ui.PrintSuccess("Code formatted and linted successfully!")
	return nil
}

func formatGoFiles() error {
	ui.PrintInfo("Formatting Go files...")
	cmd := exec.Command("gofmt", "-w", ".")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to format Go files: %w", err)
	}
	return nil
}
