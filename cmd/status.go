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
				// We must resolve the service definition from the service-map ID
				def, err := config.ResolveByID(service.ID)
				if err != nil {
					log(fmt.Sprintf("Warning: Service ID '%s' in service-map.json not found in definitions. Skipping.", service.ID))
					continue
				}
				row := checkServiceStatus(service, *def)
				rows = append(rows, row)
				log(fmt.Sprintf("Service: %s, Type: %s, Status: %s", service.ID, serviceType, row[3]))
			}
		}
	} else {
		serviceDef, err := config.Resolve(serviceShortName)
		if err != nil {
			return fmt.Errorf("failed to resolve service '%s': %w", serviceShortName, err)
		}

		// Find the matching entry in the service-map.json
		var resolvedServiceEntry *config.ServiceEntry
		for _, services := range serviceMap.Services {
			for _, service := range services {
				if service.ID == serviceDef.ID {
					resolvedServiceEntry = &service
					break
				}
			}
			if resolvedServiceEntry != nil {
				break
			}
		}

		if resolvedServiceEntry == nil {
			return fmt.Errorf("service '%s' (ID: %s) not found in your service-map.json. Run 'dex add' to add it", serviceShortName, serviceDef.ID)
		}

		row := checkServiceStatus(*resolvedServiceEntry, *serviceDef)
		rows = append(rows, row)
		log(fmt.Sprintf("Service: %s, Type: %s, Status: %s", resolvedServiceEntry.ID, serviceDef.Type, row[3]))
	}

	// Render table
	table := ui.CreateServiceTable(rows)
	table.Render()

	return nil
}

// colorizeNA colors "N/A" values dark gray, and leaves other values as-is.
func colorizeNA(value string) string {
	if value == "N/A" {
		return ui.Colorize(value, ui.ColorDarkGray)
	}
	return value
}

// checkServiceStatus acts as a dispatcher, routing to the correct status checker based on service type.
func checkServiceStatus(service config.ServiceEntry, def config.ServiceDefinition) ui.TableRow {
	// Define max lengths for columns
	const (
		maxServiceLen = 19
		maxAddressLen = 17
		maxVersionLen = 12
		maxUptimeLen  = 10
	)

	// Use the definition's shortName for display, as it's the user-facing alias
	serviceName := ui.Truncate(def.ShortName, maxServiceLen)
	address := ui.Truncate(stripProtocol(service.HTTP), maxAddressLen)

	switch def.Type {
	case "cli":
		return checkCLIStatus(serviceName)
	case "os":
		return checkCacheStatus(service, serviceName, address)
	default:
		if service.HTTP == "" {
			return ui.FormatTableRow(serviceName, colorizeNA("N/A"), colorizeNA("N/A"), colorizeStatus("N/A"), colorizeNA("N/A"), colorizeNA("N/A"), colorizeNA("N/A"), colorizeNA("N/A"))
		}
		return checkHTTPStatus(service, serviceName, address)
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
func checkCLIStatus(serviceName string) ui.TableRow {
	cmd := exec.Command("dex", "version")
	output, err := cmd.CombinedOutput()

	status := "OK"
	if err != nil {
		status = "BAD"
	}

	version := "N/A"
	outputStr := strings.TrimSpace(string(output))
	// The version command now just prints the version string.
	if err == nil && outputStr != "" {
		// Full version string: v0.3.0.main.5241102.2025-11-03-20-20-30.linux_amd64.lcr4rk
		// User wants: 0.3.0

		// Trim the 'v' prefix
		trimmedV := strings.TrimPrefix(outputStr, "v")

		// Split by '.'
		parts := strings.Split(trimmedV, ".")

		// Check if we have at least 3 parts (major, minor, patch)
		if len(parts) >= 3 {
			version = strings.Join(parts[0:3], ".") // "0.3.0"
		} else {
			// Fallback in case the version string is not as expected
			version = outputStr
		}
	}

	return ui.FormatTableRow(
		serviceName,
		colorizeNA("N/A"),
		colorizeNA(ui.Truncate(version, 12)),
		colorizeStatus(status),
		colorizeNA("N/A"),
		colorizeNA("N/A"),
		colorizeNA("N/A"),
		time.Now().Format("15:04:05"),
	)
}

// checkCacheStatus checks a cache/db service (Redis/Valkey) with an optional AUTH and a PING command.
func checkCacheStatus(service config.ServiceEntry, serviceName, address string) ui.TableRow {
	conn, err := net.DialTimeout("tcp", service.HTTP, 2*time.Second)
	if err != nil {
		return ui.FormatTableRow(serviceName, address, colorizeNA("N/A"), colorizeStatus("BAD"), colorizeNA("N/A"), colorizeNA("N/A"), colorizeNA("N/A"), time.Now().Format("15:04:05"))
	}
	defer func() { _ = conn.Close() }()

	reader := bufio.NewReader(conn)

	if service.Credentials != nil && service.Credentials.Password != "" {
		authCmd := fmt.Sprintf("AUTH %s\r\n", service.Credentials.Password)
		if _, err = conn.Write([]byte(authCmd)); err != nil {
			return ui.FormatTableRow(serviceName, address, colorizeNA("N/A"), colorizeStatus("BAD"), "Auth failed", colorizeNA("N/A"), colorizeNA("N/A"), time.Now().Format("15:04:05"))
		}
		response, err := reader.ReadString('\n')
		if err != nil || !strings.HasPrefix(response, "+OK") {
			return ui.FormatTableRow(serviceName, address, colorizeNA("N/A"), colorizeStatus("BAD"), "Auth failed", colorizeNA("N/A"), colorizeNA("N/A"), time.Now().Format("15:04:05"))
		}
	}

	if _, err = conn.Write([]byte("PING\r\n")); err != nil {
		return ui.FormatTableRow(serviceName, address, colorizeNA("N/A"), colorizeStatus("BAD"), "Ping failed", colorizeNA("N/A"), colorizeNA("N/A"), time.Now().Format("15:04:05"))
	}

	response, err := reader.ReadString('\n')
	if err != nil || !strings.HasPrefix(response, "+PONG") {
		return ui.FormatTableRow(serviceName, address, colorizeNA("N/A"), colorizeStatus("BAD"), "Ping failed", colorizeNA("N/A"), colorizeNA("N/A"), time.Now().Format("15:04:05"))
	}

	// --- Get Version ---
	version := "N/A"
	if _, err = conn.Write([]byte("INFO server\r\n")); err == nil {
		// Read the bulk string header, e.g., "$1234"
		header, err := reader.ReadString('\n')
		if err == nil && strings.HasPrefix(header, "$") {
			// Now read the actual info lines
			for {
				line, err := reader.ReadString('\n')
				if err != nil || line == "\r\n" { // End of response or error
					break
				}
				// Check for both redis_version and valkey_version
				if strings.HasPrefix(line, "redis_version:") || strings.HasPrefix(line, "valkey_version:") {
					parts := strings.Split(strings.TrimSpace(line), ":")
					if len(parts) == 2 {
						version = parts[1]
						break // Found it
					}
				}
			}
		}
	}
	// --- End Get Version ---

	return ui.FormatTableRow(serviceName, address, colorizeNA(ui.Truncate(version, 12)), colorizeStatus("OK"), colorizeNA("N/A"), colorizeNA("N/A"), colorizeNA("N/A"), time.Now().Format("15:04:05"))
}

// checkHTTPStatus checks a service via its HTTP /status endpoint
func checkHTTPStatus(service config.ServiceEntry, serviceName, address string) ui.TableRow {
	statusURL := "http://" + service.HTTP + "/status"
	resp, err := http.Get(statusURL)
	if err != nil {
		return ui.FormatTableRow(serviceName, address, colorizeNA("N/A"), colorizeStatus("BAD"), colorizeNA("N/A"), colorizeNA("N/A"), colorizeNA("N/A"), time.Now().Format("15:04:05"))
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ui.FormatTableRow(serviceName, address, colorizeNA("N/A"), colorizeStatus("BAD"), colorizeNA("N/A"), colorizeNA("N/A"), colorizeNA("N/A"), time.Now().Format("15:04:05"))
	}

	var statusResp health.StatusResponse
	if err := json.Unmarshal(body, &statusResp); err != nil {
		return ui.FormatTableRow(serviceName, address, colorizeNA("N/A"), colorizeStatus("BAD"), colorizeNA("N/A"), colorizeNA("N/A"), colorizeNA("N/A"), time.Now().Format("15:04:05"))
	}

	uptime := colorizeNA(ui.Truncate(formatUptime(time.Duration(statusResp.Uptime)*time.Second), 10))
	goroutines := colorizeNA(fmt.Sprintf("%d", statusResp.Metrics.Goroutines))
	mem := colorizeNA(fmt.Sprintf("%.2f", statusResp.Metrics.MemoryAllocMB))

	// Use the serviceName (short-name) from the definition, not the one from the /status response
	return ui.FormatTableRow(
		serviceName,
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
