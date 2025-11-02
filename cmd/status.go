package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/health"
	"github.com/EasterCompany/dex-cli/ui"
)

// Status checks the health of one or all services
func Status(serviceName string) error {
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
		row := checkServiceStatus(service)
		rows = append(rows, row)
	}

	// Render table
	table := ui.CreateServiceTable(rows)
	fmt.Print(ui.RenderTable(table))

	return nil
}

// checkServiceStatus checks the status of a service based on its type
func checkServiceStatus(service config.ServiceEntry) ui.TableRow {
	// Handle dex-cli specially - check if command exists
	if service.ID == "dex-cli" {
		return checkCLIStatus(service)
	}

	// Handle services with no address (skipped)
	if service.Addr == "" {
		return ui.FormatTableRow(
			service.ID,
			"N/A",
			"N/A",
			"SKIPPED",
			"N/A",
			"N/A",
		)
	}

	// All other services use HTTP status endpoint
	return checkHTTPStatus(service)
}

// checkCLIStatus checks if a CLI tool is installed and working
func checkCLIStatus(service config.ServiceEntry) ui.TableRow {
	cmd := exec.Command("dex", "version")
	output, err := cmd.CombinedOutput()

	if err != nil {
		return ui.FormatTableRow(
			service.ID,
			"local",
			"N/A",
			"NOT INSTALLED",
			"N/A",
			time.Now().Format("15:04:05"),
		)
	}

	// Parse version from output (format: "dex version X.X.X")
	version := "N/A"
	outputStr := strings.TrimSpace(string(output))
	if strings.HasPrefix(outputStr, "dex version ") {
		version = strings.TrimPrefix(outputStr, "dex version ")
	}

	return ui.FormatTableRow(
		service.ID,
		"local",
		version,
		"INSTALLED",
		"N/A",
		time.Now().Format("15:04:05"),
	)
}

// checkHTTPStatus checks a service via its HTTP /status endpoint
func checkHTTPStatus(service config.ServiceEntry) ui.TableRow {
	statusURL := strings.TrimSuffix(service.Addr, "/") + "/status"
	resp, err := http.Get(statusURL)
	if err != nil {
		return ui.FormatTableRow(
			service.ID,
			service.Addr,
			"N/A",
			"DOWN",
			"N/A",
			time.Now().Format("15:04:05"),
		)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ui.FormatTableRow(
			service.ID,
			service.Addr,
			"N/A",
			"ERROR",
			"N/A",
			time.Now().Format("15:04:05"),
		)
	}

	var statusResp health.StatusResponse
	if err := json.Unmarshal(body, &statusResp); err != nil {
		return ui.FormatTableRow(
			service.ID,
			service.Addr,
			"N/A",
			"INVALID",
			"N/A",
			time.Now().Format("15:04:05"),
		)
	}

	uptime := formatUptime(time.Duration(statusResp.Uptime) * time.Second)
	return ui.FormatTableRow(
		statusResp.Service,
		service.Addr,
		statusResp.Version,
		statusResp.Status,
		uptime,
		time.Unix(statusResp.Timestamp, 0).Format("15:04:05"),
	)
}
