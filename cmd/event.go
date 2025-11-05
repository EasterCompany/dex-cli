package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/ui"
	"github.com/EasterCompany/dex-cli/utils"
)

// Event provides commands to interact with the dex-event-service
func Event(args []string) error {
	def, err := config.Resolve("event")
	if err != nil {
		return err
	}

	status, _, err := utils.GetHTTPBody(def.GetHTTP("/service"))
	if err != nil {
		return fmt.Errorf("failed to get event service status: %w", err)
	}

	// --- Debugging Step 1: Print Raw Response ---
	ui.PrintHeader("1. Raw Response")
	ui.PrintRaw(string(status) + "\n")

	// --- Debugging Step 2: Print Formatted Response ---
	var v interface{}
	if err := json.Unmarshal(status, &v); err == nil {
		if formatted, err := json.MarshalIndent(v, "", "  "); err == nil {
			ui.PrintHeader("2. Formatted Response")
			ui.PrintRaw(string(formatted) + "\n")
		}
	}

	return nil
}
