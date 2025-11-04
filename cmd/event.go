package cmd

import (
	"fmt"
	"io"
	"net/http"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/ui"
)

// Event provides commands to interact with the dex-event-service
func Event(args []string) error {
	serviceMap, err := config.LoadServiceMapConfig()
	if err != nil {
		return fmt.Errorf("failed to load service-map.json: %w", err)
	}

	var eventService *config.ServiceEntry
	for _, service := range serviceMap.Services["cs"] {
		if service.ID == "dex-event-service" {
			eventService = &service
			break
		}
	}

	if eventService == nil {
		return fmt.Errorf("dex-event-service not found in service-map.json")
	}

	if len(args) == 0 {
		ui.PrintInfo("Event command. Available subcommands: [status, health]")
		return nil
	}

	switch args[0] {
	case "status":
		return eventStatus(eventService)
	case "health":
		return eventHealth(eventService)
	default:
		ui.PrintInfo(fmt.Sprintf("Unknown event subcommand: %s", args[0]))
		ui.PrintInfo("Available subcommands: [status, health]")
		return nil
	}
}

func eventStatus(service *config.ServiceEntry) error {
	url := fmt.Sprintf("http://%s:%s/status", service.Domain, service.Port)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to get event service status: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read event service status response: %w", err)
	}

	ui.PrintCodeBlockFromBytes(body, "response.json", "json")
	return nil
}

func eventHealth(service *config.ServiceEntry) error {
	url := fmt.Sprintf("http://%s:%s/health", service.Domain, service.Port)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to get event service health: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read event service health response: %w", err)
	}

	ui.PrintCodeBlockFromBytes(body, "response.json", "json")
	return nil
}
