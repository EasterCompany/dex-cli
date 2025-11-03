package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/health"
	"github.com/EasterCompany/dex-cli/ui"
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
		// 1. Resolve the alias to a full service definition
		def, err := config.Resolve(serviceShortName)
		if err != nil {
			return err
		}

		// 2. Find the corresponding entry in service-map.json
		var resolvedServiceEntry *config.ServiceEntry
		var serviceType string
		for sType, services := range serviceMap.Services {
			for _, service := range services {
				if service.ID == def.ID {
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
			// Special case: 'dex-cli' is in ServiceDefinitions but not in service-map.json by default
			if def.Type == "cli" {
				row := checkServiceStatus(config.ServiceEntry{ID: def.ID}, def.Type)
				rows = append(rows, row)
			} else {
				return fmt.Errorf("service '%s' (%s) not found in service-map.json. Run 'dex add %s'?", serviceShortName, def.ID, serviceShortName)
			}
		} else {
			row := checkServiceStatus(*resolvedServiceEntry, serviceType)
			rows = append(rows, row)
			log(fmt.Sprintf("Service: %s, Type: %s, Status: %s", resolvedServiceEntry.ID, serviceType, row[3]))
		}
	}

	// Render table
	table := ui.CreateServiceTable(rows)
	table.Render()

	return nil
}

// checkServiceStatus acts as a dispatcher, routing to the correct status checker based on service type.
func checkServiceStatus(service config.ServiceEntry, serviceType string) ui.TableRow {
	// Define max lengths for columns
	const (
		maxServiceLen = 19
		maxAddressLen = 17
		maxVersionLen = 12
		maxUptimeLen  = 10
	)

	serviceID := ui.Truncate(service.ID, maxServiceLen)
	address := ui.Truncate(stripProtocol(service.HTTP), maxAddressLen)

	switch serviceType {
	case "cli":
		return checkCLIStatus(service, serviceID)
	case "os":
		return checkCacheStatus(service, serviceID, address)
	default:
		if service.HTTP == "" {
			return ui.FormatTableRow(serviceID, colorizeNA("N/A"), colorizeNA("N/A"), colorizeStatus("N/A"), colorizeNA("N/A"), colorizeNA("N/A"), colorizeNA("N/A"), time.Now().Format("15:04:05"))
		}
		return checkHTTPStatus(service, serviceID, address)
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
func checkCLIStatus(service config.ServiceEntry, serviceID string) ui.TableRow {
	cmd := exec.Command("dex", "version")
	output, err := cmd.CombinedOutput()

	status := "OK"
	if err != nil {
		status = "BAD"
	}

	version := "N/A"
	// Get the last line of output, in case of update messages
	outputLines := strings.Split(strings.TrimSpace(string(output)), "\n")
	outputStr := outputLines[len(outputLines)-1]

	// Parse the version string, which is complex
	// v0.3.0.main.5241102.2025-11-03-20-20-30.linux_amd64.lcr4rk
	// We want to show: v0.3.0.5241102
	parts := strings.Split(outputStr, ".")
	if len(parts) >= 4 && strings.HasPrefix(parts[0], "v") {
		version = fmt.Sprintf("%s.%s.%s.%s", parts[0], parts[1], parts[2], parts[4])
	} else if strings.Contains(outputStr, " | Easter Company™") {
		version = strings.Split(outputStr, " | Easter Company™")[0]
	}

	return ui.FormatTableRow(
		serviceID,
		"local",
		colorizeNA(ui.Truncate(version, 12)),
		colorizeStatus(status),
		colorizeNA("N/A"),
		colorizeNA("N/A"),
		colorizeNA("N/A"),
		time.Now().Format("15:04:05"),
	)
}

// checkCacheStatus checks a cache/db service (Redis/Valkey) with an optional AUTH and a PING command.
func checkCacheStatus(service config.ServiceEntry, serviceID, address string) ui.TableRow {
	conn, err := net.DialTimeout("tcp", service.HTTP, 2*time.Second)
	if err != nil {
		return ui.FormatTableRow(serviceID, address, colorizeNA("N/A"), colorizeStatus("BAD"), colorizeNA("N/A"), colorizeNA("N/A"), colorizeNA("N/A"), time.Now().Format("15:04:05"))
	}
	defer func() { _ = conn.Close() }()

	reader := bufio.NewReader(conn)

	if service.Credentials != nil && service.Credentials.Password != "" {
		authCmd := fmt.Sprintf("AUTH %s\r\n", service.Credentials.Password)
		if _, err = conn.Write([]byte(authCmd)); err != nil {
			return ui.FormatTableRow(serviceID, address, colorizeNA("N/A"), colorizeStatus("BAD"), "Auth failed", colorizeNA("N/A"), colorizeNA("N/A"), time.Now().Format("15:04:05"))
		}
		response, err := reader.ReadString('\n')
		if err != nil || !strings.HasPrefix(response, "+OK") {
			return ui.FormatTableRow(serviceID, address, colorizeNA("N/A"), colorizeStatus("BAD"), "Auth failed", colorizeNA("N/A"), colorizeNA("N/A"), time.Now().Format("15:04:05"))
		}
	}

	if _, err = conn.Write([]byte("PING\r\n")); err != nil {
		return ui.FormatTableRow(serviceID, address, colorizeNA("N/A"), colorizeStatus("BAD"), "Ping failed", colorizeNA("N/A"), colorizeNA("N/A"), time.Now().Format("15:04:05"))
	}

	response, err := reader.ReadString('\n')
	if err != nil || !strings.HasPrefix(response, "+PONG") {
		return ui.FormatTableRow(serviceID, address, colorizeNA("N/A"), colorizeStatus("BAD"), "Ping failed", colorizeNA("N/A"), colorizeNA("N/A"), time.Now().Format("15:04:05"))
	}

	return ui.FormatTableRow(serviceID, address, colorizeNA("N/A"), colorizeStatus("OK"), colorizeNA("N/A"), colorizeNA("N/A"), colorizeNA("N/A"), time.Now().Format("15:04:05"))
}

// checkHTTPStatus checks a service via its HTTP /status endpoint
func checkHTTPStatus(service config.ServiceEntry, serviceID, address string) ui.TableRow {
	// Construct the URL, assuming HTTP if no scheme is present
	statusURL := service.HTTP
	if !strings.HasPrefix(statusURL, "http://") && !strings.HasPrefix(statusURL, "https://") {
		statusURL = "http://" + statusURL
	}
	// Add /status endpoint
	statusURL = strings.TrimSuffix(statusURL, "/") + "/status"

	client := http.Client{
		Timeout: 2 * time.Second,
	}
	resp, err := client.Get(statusURL)
	if err != nil {
		return ui.FormatTableRow(serviceID, address, colorizeNA("N/A"), colorizeStatus("BAD"), colorizeNA("N/A"), colorizeNA("N/A"), colorizeNA("N/A"), time.Now().Format("15:04:05"))
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ui.FormatTableRow(serviceID, address, colorizeNA("N/A"), colorizeStatus("BAD"), colorizeNA("N/A"), colorizeNA("N/A"), colorizeNA("N/A"), time.Now().Format("15:04:05"))
	}

	var statusResp health.StatusResponse
	if err := json.Unmarshal(body, &statusResp); err != nil {
		// If JSON fails, it might be a non-Go service. Check for 200 OK.
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return ui.FormatTableRow(serviceID, address, colorizeNA("N/A"), colorizeStatus("OK"), colorizeNA("N/A"), colorizeNA("N/A"), colorizeNA("N/A"), time.Now().Format("15:04:05"))
		}
		return ui.FormatTableRow(serviceID, address, colorizeNA("N/A"), colorizeStatus("BAD"), colorizeNA("N/A"), colorizeNA("N/A"), colorizeNA("N/A"), time.Now().Format("15:04:05"))
	}

	uptime := ui.Truncate(formatUptime(time.Duration(statusResp.Uptime)*time.Second), 10)
	goroutines := fmt.Sprintf("%d", statusResp.Metrics.Goroutines)
	mem := fmt.Sprintf("%.2f", statusResp.Metrics.MemoryAllocMB)

	return ui.FormatTableRow(
		ui.Truncate(statusResp.Service, 19),
		address,
		colorizeNA(ui.Truncate(statusResp.Version, 12)),
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

// colorizeNA colors "N/A" values dark gray, and leaves other values as-is.
func colorizeNA(value string) string {
	if value == "N/A" {
		return ui.Colorize(value, ui.ColorDarkGray)
	}
	return value
}
