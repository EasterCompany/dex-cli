package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/health"
	"github.com/EasterCompany/dex-cli/ui"
)

// Watch provides a live dashboard of all service statuses
func Watch() error {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if err := refreshWatchDashboard(); err != nil {
			ui.PrintError(fmt.Sprintf("Error refreshing dashboard: %v", err))
		}
	}

	return nil
}

func clearScreen() {
	cmd := exec.Command("clear") // or "cls" on Windows
	cmd.Stdout = os.Stdout
	cmd.Run()
}

func refreshWatchDashboard() error {
	clearScreen()
	ui.PrintTitle("DEXTER LIVE WATCH DASHBOARD")
	fmt.Println("Press Ctrl+C to exit")
	fmt.Println()

	serviceMap, err := config.LoadServiceMap()
	if err != nil {
		return fmt.Errorf("failed to load service map: %w", err)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "SERVICE\tVERSION\tSTATUS\tUPTIME\tLAST CHECK")
	fmt.Fprintln(w, "-------\t-------\t------\t------\t----------")

	for _, services := range serviceMap.Services {
		for _, service := range services {
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
			client := http.Client{Timeout: 1 * time.Second}
			resp, err := client.Get(statusURL)
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
	}
	w.Flush()

	return nil
}
