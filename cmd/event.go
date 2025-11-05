package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/ui"
)

// Event provides commands to interact with the dex-event-service
func Event(args []string) error {
	def, err := config.Resolve("event")
	if err != nil {
		return err
	}

	report, err := GetServiceReport(*def)
	if err != nil {
		return fmt.Errorf("failed to get event service report: %w", err)
	}

	reportJSON, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal service report: %w", err)
	}

	ui.PrintCodeBlockFromBytes(reportJSON, "response.json", "json")
	return nil
}
