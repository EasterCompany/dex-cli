package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/EasterCompany/dex-cli/ui"
)

// Test runs the test suite
func Test(args []string) error {
	fmt.Println(ui.RenderTitle("RUNNING TESTS"))

	if err := runGoTests(); err != nil {
		return err
	}

	ui.PrintSuccess("All tests passed!")
	return nil
}

func runGoTests() error {
	ui.PrintInfo("Running Go tests...")
	cmd := exec.Command("go", "test", "./...")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run Go tests: %w", err)
	}
	return nil
}
