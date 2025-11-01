package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/health"
	"github.com/EasterCompany/dex-cli/ui"
)

// Status checks the health of one or all services
func Status(serviceName string) error {
	fmt.Println(ui.RenderTitle("DEXTER STATUS"))
	fmt.Println()

	// Load the service map
	serviceMap, err := config.LoadServiceMap()
	if err != nil {
		return fmt.Errorf("failed to load service map: %w", err)
	}

	var servicesToCheck []config.ServiceEntry

	if serviceName == "all" || serviceName == "" {
		// Check all services
		for _, services := range serviceMap.Services {
			servicesToCheck = append(servicesToCheck, services...)
		}
	} else {
		// Check a specific service
		found := false
		for _, services := range serviceMap.Services {
			for _, service := range services {
				if service.ID == serviceName {
					servicesToCheck = append(servicesToCheck, service)
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			return fmt.Errorf("service '%s' not found in service-map.json", serviceName)
		}
	}

	// Build table rows
	var rows []ui.TableRow
	for _, service := range servicesToCheck {
		if service.Addr == "" {
			rows = append(rows, ui.FormatTableRow(
				service.ID,
				"N/A",
				"SKIPPED",
				"N/A",
				"N/A",
			))
			continue
		}

		statusURL := strings.TrimSuffix(service.Addr, "/") + "/status"
		resp, err := http.Get(statusURL)
		if err != nil {
			rows = append(rows, ui.FormatTableRow(
				service.ID,
				"N/A",
				"DOWN",
				"N/A",
				time.Now().Format("15:04:05"),
			))
			continue
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			rows = append(rows, ui.FormatTableRow(
				service.ID,
				"N/A",
				"ERROR",
				"N/A",
				time.Now().Format("15:04:05"),
			))
			continue
		}

		var statusResp health.StatusResponse
		if err := json.Unmarshal(body, &statusResp); err != nil {
			rows = append(rows, ui.FormatTableRow(
				service.ID,
				"N/A",
				"INVALID RESP",
				"N/A",
				time.Now().Format("15:04:05"),
			))
			continue
		}

		uptime := formatUptime(time.Duration(statusResp.Uptime) * time.Second)
		rows = append(rows, ui.FormatTableRow(
			statusResp.Service,
			statusResp.Version,
			statusResp.Status,
			uptime,
			time.Unix(statusResp.Timestamp, 0).Format("15:04:05"),
		))
	}

	// Render table
	table := ui.CreateServiceTable(rows)
	fmt.Print(ui.RenderTable(table))

	return nil
}
