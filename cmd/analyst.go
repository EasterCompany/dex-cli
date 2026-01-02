package cmd

import (
	"fmt"
	"net/http"
	"time"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/ui"
)

func Analyst(args []string) error {
	command := "run"
	if len(args) > 0 {
		command = args[0]
	}

	def, err := config.Resolve("event")
	if err != nil {
		return fmt.Errorf("failed to resolve event service: %w", err)
	}

	switch command {
	case "run":
		ui.PrintHeader("Analyst Biographical Synthesis")
		ui.PrintInfo("Triggering Synthesis Protocol...")

		url := def.GetHTTP("/analyzer/run")
		req, _ := http.NewRequest("POST", url, nil)
		client := &http.Client{Timeout: 10 * time.Second}

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("failed to trigger analyst: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("analyst run failed with status: %d", resp.StatusCode)
		}

		ui.PrintSuccess("Analyst synthesis protocol triggered successfully in background.")
		ui.PrintInfo("You can monitor progress in the dashboard or service logs.")

	case "reset":
		ui.PrintHeader("Analyst Reset")
		ui.PrintInfo("Resetting Analyst protocols...")

		url := def.GetHTTP("/analyzer/reset")
		req, _ := http.NewRequest("POST", url, nil)
		client := &http.Client{Timeout: 10 * time.Second}

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("failed to reset analyst: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("analyst reset failed with status: %d", resp.StatusCode)
		}

		ui.PrintSuccess("Analyst protocols reset successfully.")

	default:
		return fmt.Errorf("unknown analyst command: %s", command)
	}

	return nil
}
