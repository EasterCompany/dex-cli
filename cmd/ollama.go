package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/EasterCompany/dex-cli/utils"
)

func Ollama(args []string) error {
	if len(args) == 1 && args[0] == "pull" {
		return utils.PullHardcodedModels()
	}

	ollamaPath, err := exec.LookPath("ollama")
	if err != nil {
		return fmt.Errorf("failed to find 'ollama' executable in system PATH: %w\nPlease ensure Ollama is installed and its location is included in your PATH", err)
	}

	cmd := exec.Command(ollamaPath, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("ollama command failed: %w", err)
	}

	return nil
}
