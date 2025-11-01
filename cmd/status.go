package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/health"
	"github.com/EasterCompany/dex-cli/ui"
)

// Status checks the health of one or all services
func Status(serviceName string) error {
	ui.PrintTitle("DEXTER STATUS COMMAND")

	// Load the service map
	ui.PrintSectionTitle("LOADING SERVICE MAP")
	serviceMap, err := config.LoadServiceMap()
	if err != nil {
		return fmt.Errorf("failed to load service map: %w", err)
	}
	ui.PrintSuccess("Service map loaded")

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

	ui.PrintSectionTitle("SERVICE STATUS")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "SERVICE\tVERSION\tSTATUS\tUPTIME\tLAST CHECK")
	fmt.Fprintln(w, "-------\t-------\t------\t------\t----------")

	for _, service := range servicesToCheck {
		if service.Addr == "" {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				ui.Colorize(service.ID, ui.ColorYellow),
				"N/A",
				ui.Colorize("SKIPPED", ui.ColorYellow),
				"N/A",
				"N/A")
			continue
		}

		statusURL := strings.TrimSuffix(service.Addr, "/") + "/status"
		resp, err := http.Get(statusURL)
		if err != nil {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				ui.Colorize(service.ID, ui.ColorRed),
				"N/A",
				ui.Colorize("DOWN", ui.ColorRed),
				"N/A",
				time.Now().Format("15:04:05"))
			continue
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				ui.Colorize(service.ID, ui.ColorRed),
				"N/A",
				ui.Colorize("ERROR", ui.ColorRed),
				"N/A",
				time.Now().Format("15:04:05"))
			continue
		}

		var statusResp health.StatusResponse
		if err := json.Unmarshal(body, &statusResp); err != nil {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				ui.Colorize(service.ID, ui.ColorRed),
				"N/A",
				ui.Colorize("INVALID RESP", ui.ColorRed),
				"N/A",
				time.Now().Format("15:04:05"))
			continue
		}

		statusColor := ui.ColorGreen
		if statusResp.Status == "degraded" {
			statusColor = ui.ColorYellow
		} else if statusResp.Status == "unhealthy" {
			statusColor = ui.ColorRed
		}

		uptime := formatUptime(time.Duration(statusResp.Uptime) * time.Second)
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			ui.Colorize(statusResp.Service, ui.ColorWhite),
			statusResp.Version,
			ui.Colorize(strings.ToUpper(statusResp.Status), statusColor),
			uptime,
			time.Unix(statusResp.Timestamp, 0).Format("15:04:05"))
	}
	w.Flush()

	return nil
}
