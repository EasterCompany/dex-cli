package cmd

import (
	"fmt"
	"os"
	"os/exec"
)

// Ollama serves as a proxy for the system's actual 'ollama' executable.
// It executes the system command with all provided arguments and pipes
// the standard output and error directly back to the user.
func Ollama(args []string) error {
	// 1. Locate the 'ollama' executable in the system's PATH.
	ollamaPath, err := exec.LookPath("ollama")
	if err != nil {
		// If the executable is not found, inform the user clearly.
		return fmt.Errorf("failed to find 'ollama' executable in system PATH: %w\nPlease ensure Ollama is installed and its location is included in your PATH", err)
	}

	// 2. Build the command. The first argument is the path to the executable,
	// followed by all user-provided arguments.
	cmd := exec.Command(ollamaPath, args...)

	// 3. Proxy I/O: Set the command's standard input, output, and error streams
	// to match the current process's streams. This ensures the user sees the
	// native 'ollama' output and can interact with it (e.g., streaming).
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// 4. Execute the command and wait for it to finish.
	err = cmd.Run()
	if err != nil {
		// If the command ran but returned a non-zero exit code (e.g., 'ollama' reported an error),
		// exec.Run() will return an *ExitError. We return this error.
		return fmt.Errorf("ollama command failed: %w", err)
	}

	return nil
}
