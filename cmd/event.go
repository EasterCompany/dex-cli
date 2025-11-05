package cmd

import (
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

	ui.PrintCodeBlockFromBytes(status, "event-service", "json")
	return nil
}
