package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os/exec"
	"strings"
	"time"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/health"
	"github.com/EasterCompany/dex-cli/ui"
	"net/http"
)

// Status checks the health of one or all services
func Status(serviceShortName string) error {
	logFile, err := config.LogFile()
	if err != nil {
		return fmt.Errorf("failed to get log file: %w", err)
	}
	defer func() { _ = logFile.Close() }()

	log := func(message string) {
		_, _ = fmt.Fprintln(logFile, message)
	}

	log(fmt.Sprintf("Checking status for service: %s", serviceShortName))

	serviceMap, err := config.LoadServiceMapConfig()
	if err != nil {
		return fmt.Errorf("failed to load service map: %w", err)
	}

	var rows []ui.TableRow

	if serviceShortName == "all" || serviceShortName == "" {
		// Iterate in the desired order for consistent output
		serviceTypes := []string{"cli", "fe", "cs", "be", "th", "os"}
		for _, serviceType := range serviceTypes {
			services := serviceMap.Services[serviceType]
			for _, service := range services {
				row := checkServiceStatus(service, serviceType)
				rows = append(rows, row)
				log(fmt.Sprintf("Service: %s, Type: %s, Status: %s", service.ID, serviceType, row[3]))
			}
		}
	} else {
		projectDirName, err := config.ResolveProjectDirService(serviceShortName)
		if err != nil {
			return fmt.Errorf("failed to resolve service '%s': %w", serviceShortName, err)
		}

		var resolvedServiceEntry *config.ServiceEntry
		var serviceType string
		for sType, services := range serviceMap.Services {
			for _, service := range services {
				if service.ID == projectDirName {
					resolvedServiceEntry = &service
					serviceType = sType
					break
				}
			}
			if resolvedServiceEntry != nil {
				break
			}
		}

		if resolvedServiceEntry == nil {
			return fmt.Errorf("service '%s' (resolved to '%s') not found in service-map.json", serviceShortName, projectDirName)
		}

		row := checkServiceStatus(*resolvedServiceEntry, serviceType)
		rows = append(rows, row)
		log(fmt.Sprintf("Service: %s, Type: %s, Status: %s", resolvedServiceEntry.ID, serviceType, row[3]))
	}

	// Render table
	table := ui.CreateServiceTable(rows)
	table.Render()

	return nil
}

// checkServiceStatus acts as a dispatcher, routing to the correct status checker based on service type.
func checkServiceStatus(service config.ServiceEntry, serviceType string) ui.TableRow {
	// Strip protocol and truncate address
	address := stripProtocol(service.HTTP)
	if len(address) >= 18 {
		address = address[:14] + "..."
	}

	switch serviceType {
	case "cli":
		return checkCLIStatus(service)
	case "os":
		// All 'os' services are currently cache/db instances
		return checkCacheStatus(service, address)
	default:
		// All other types are standard HTTP services
		if service.HTTP == "" {
			return ui.FormatTableRow(service.ID, "N/A", "N/A", "N/A", "N/A", "N/A", "N/A", "N/A")
		}
		return checkHTTPStatus(service, address)
	}
}

// stripProtocol removes common URL schemes from a string.
func stripProtocol(address string) string {
	address = strings.TrimPrefix(address, "http://")
	address = strings.TrimPrefix(address, "https://")
	address = strings.TrimPrefix(address, "ws://")
	address = strings.TrimPrefix(address, "wss://")
	return address
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
			"BAD",
			"N/A",
			"N/A",
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
		colorizeStatus("OK"),
		"N/A",
		"N/A",
		"N/A",
		time.Now().Format("15:04:05"),
	)
}

// checkCacheStatus checks a cache/db service (Redis/Valkey) with an optional AUTH and a PING command.
func checkCacheStatus(service config.ServiceEntry, address string) ui.TableRow {
	conn, err := net.DialTimeout("tcp", service.HTTP, 2*time.Second)
	if err != nil {
		return ui.FormatTableRow(service.ID, address, "N/A", colorizeStatus("BAD"), "N/A", "N/A", "N/A", time.Now().Format("15:04:05"))
	}
	defer func() { _ = conn.Close() }()

	reader := bufio.NewReader(conn)

	// --- Handle Authentication ---
	if service.Credentials != nil && service.Credentials.Password != "" {
		authCmd := fmt.Sprintf("AUTH %s\r\n", service.Credentials.Password)
		if _, err = conn.Write([]byte(authCmd)); err != nil {
			return ui.FormatTableRow(service.ID, address, "N/A", colorizeStatus("BAD"), "Auth failed", "N/A", "N/A", time.Now().Format("15:04:05"))
		}
		response, err := reader.ReadString('\n')
		if err != nil || !strings.HasPrefix(response, "+OK") {
			return ui.FormatTableRow(service.ID, address, "N/A", colorizeStatus("BAD"), "Auth failed", "N/A", "N/A", time.Now().Format("15:04:05"))
		}
	}

	// --- Send PING ---
	if _, err = conn.Write([]byte("PING\r\n")); err != nil {
		return ui.FormatTableRow(service.ID, address, "N/A", colorizeStatus("BAD"), "Ping failed", "N/A", "N/A", time.Now().Format("15:04:05"))
	}

	// --- Read PONG response ---
	response, err := reader.ReadString('\n')
	if err != nil || !strings.HasPrefix(response, "+PONG") {
		return ui.FormatTableRow(service.ID, address, "N/A", colorizeStatus("BAD"), "Ping failed", "N/A", "N/A", time.Now().Format("15:04:05"))
	}

	// PING successful
	return ui.FormatTableRow(service.ID, address, "N/A", colorizeStatus("OK"), "N/A", "N/A", "N/A", time.Now().Format("15:04:05"))
}

// checkHTTPStatus checks a service via its HTTP /status endpoint
func checkHTTPStatus(service config.ServiceEntry, address string) ui.TableRow {
	statusURL := "http://" + strings.TrimPrefix(service.HTTP, "http://") + "/status"
	resp, err := http.Get(statusURL)
	if err != nil {
		return ui.FormatTableRow(
			service.ID,
			address,
			"N/A",
			colorizeStatus("BAD"),
			"N/A",
			"N/A",
			"N/A",
			time.Now().Format("15:04:05"),
		)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ui.FormatTableRow(
			service.ID,
			address,
			"N/A",
			colorizeStatus("BAD"),
			"N/A",
			"N/A",
			"N/A",
			time.Now().Format("15:04:05"),
		)
	}

	var statusResp health.StatusResponse
	if err := json.Unmarshal(body, &statusResp); err != nil {
		return ui.FormatTableRow(
			service.ID,
			address,
			"N/A",
			colorizeStatus("BAD"),
			"N/A",
			"N/A",
			"N/A",
			time.Now().Format("15:04:05"),
		)
	}

	uptime := formatUptime(time.Duration(statusResp.Uptime) * time.Second)
	goroutines := fmt.Sprintf("%d", statusResp.Metrics.Goroutines)
	mem := fmt.Sprintf("%.2f", statusResp.Metrics.MemoryAllocMB)

	return ui.FormatTableRow(
		statusResp.Service,
		address,
		statusResp.Version,
		colorizeStatus(statusResp.Status),
		uptime,
		goroutines,
		mem,
		time.Unix(statusResp.Timestamp, 0).Format("15:04:05"),
	)
}

// colorizeStatus applies color coding to the status string.
func colorizeStatus(status string) string {
	switch status {
	case "OK":
		return ui.Colorize(status, ui.ColorGreen)
	case "BAD":
		return ui.Colorize(status, ui.ColorBrightRed)
	case "N/A":
		return ui.Colorize(status, ui.ColorDarkGray)
	default:
		return status
	}
}
