// cmd/status.go
package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os/exec"
	"regexp"
	"strconv"
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
		allServices := config.GetAllServices()
		for _, service := range allServices {
			row := checkServiceStatus(service, serviceMap)
			rows = append(rows, row)
			log(fmt.Sprintf("Service: %s, Type: %s, Status: %s", service.ID, service.Type, row[3]))
		}
	} else {
		def, err := config.Resolve(serviceShortName)
		if err != nil {
			return fmt.Errorf("failed to resolve service '%s': %w", serviceShortName, err)
		}

		row := checkServiceStatus(*def, serviceMap)
		rows = append(rows, row)
		log(fmt.Sprintf("Service: %s, Type: %s, Status: %s", def.ID, def.Type, row[3]))
	}

	// Render table
	table := ui.CreateServiceTable(rows)
	table.Render()

	return nil
}

// checkServiceStatus acts as a dispatcher, routing to the correct status checker based on service type.
func checkServiceStatus(service config.ServiceDefinition, serviceMap *config.ServiceMapConfig) ui.TableRow {
	// Define max lengths for columns
	const (
		maxServiceLen = 19
		maxAddressLen = 17
		maxVersionLen = 12
		maxUptimeLen  = 10
	)

	serviceID := ui.Truncate(service.ID, maxServiceLen)
	address := ui.Truncate(service.GetHost(), maxAddressLen)

	// Check if service is installed by checking service-map.json
	isInstalled := isServiceInMap(service, serviceMap)

	switch service.Type {
	case "cli":
		return checkCLIStatus(service, serviceID)
	case "os":
		return checkCacheStatus(service, serviceID, address)
	default:
		if !isInstalled {
			return ui.FormatTableRow(
				serviceID,
				colorizeNA(address),
				colorizeNA("N/A"),
				colorizeStatus("N/A"),
				colorizeNA("N/A"),
				colorizeNA("N/A"),
				colorizeNA("N/A"),
				time.Now().Format("15:04:05"),
			)
		}
		return checkHTTPStatus(service, serviceID, address)
	}
}

// checkCLIStatus checks if a CLI tool is installed and working
func checkCLIStatus(service config.ServiceDefinition, serviceID string) ui.TableRow {
	cmd := exec.Command("dex", "version")
	output, err := cmd.CombinedOutput()

	status := "OK"
	if err != nil {
		status = "BAD"
	}

	version := "N/A"
	outputStr := strings.TrimSpace(string(output))
	// Example: v0.3.0.main.5241102.2025-11-03-20-20-30.linux_amd64.lcr4rk
	re := regexp.MustCompile(`v([0-9]+\.[0-9]+\.[0-9]+)`)
	matches := re.FindStringSubmatch(outputStr)

	if len(matches) > 1 {
		version = matches[1] // Get the captured group
	}

	return ui.FormatTableRow(
		serviceID,
		colorizeNA("N/A"),
		colorizeNA(ui.Truncate(version, 12)),
		colorizeStatus(status),
		colorizeNA("N/A"),
		colorizeNA("N/A"),
		colorizeNA("N/A"),
		time.Now().Format("15:04:05"),
	)
}

// colorizeNA colors "N/A" values dark gray, and leaves other values as-is.
func colorizeNA(value string) string {
	if value == "N/A" {
		return ui.Colorize(value, ui.ColorDarkGray)
	}
	return value
}

// checkCacheStatus checks a cache/db service (Redis/Valkey) with an optional AUTH and a PING command.
func checkCacheStatus(service config.ServiceDefinition, serviceID, address string) ui.TableRow {
	conn, err := net.DialTimeout("tcp", service.GetHost(), 2*time.Second)
	if err != nil {
		return ui.FormatTableRow(serviceID, address, colorizeNA("N/A"), colorizeStatus("BAD"), colorizeNA("N/A"), colorizeNA("N/A"), colorizeNA("N/A"), time.Now().Format("15:04:05"))
	}
	defer func() { _ = conn.Close() }()

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	// Handle AUTH
	if service.Credentials != nil && service.Credentials.Password != "" {
		authCmd := fmt.Sprintf("AUTH %s\r\n", service.Credentials.Password)
		if _, err = writer.WriteString(authCmd); err != nil {
			return ui.FormatTableRow(serviceID, address, colorizeNA("N/A"), colorizeStatus("BAD"), colorizeNA("N/A"), colorizeNA("N/A"), colorizeNA("N/A"), time.Now().Format("15:04:05"))
		}
		writer.Flush()
		response, err := reader.ReadString('\n')
		if err != nil || !strings.HasPrefix(response, "+OK") {
			return ui.FormatTableRow(serviceID, address, colorizeNA("N/A"), colorizeStatus("BAD"), colorizeNA("Auth"), colorizeNA("N/A"), colorizeNA("N/A"), time.Now().Format("15:04:05"))
		}
	}

	// Send PING
	if _, err = writer.WriteString("PING\r\n"); err != nil {
		return ui.FormatTableRow(serviceID, address, colorizeNA("N/A"), colorizeStatus("BAD"), colorizeNA("Ping"), colorizeNA("N/A"), colorizeNA("N/A"), time.Now().Format("15:04:05"))
	}
	writer.Flush()
	response, err := reader.ReadString('\n')
	if err != nil || !strings.HasPrefix(response, "+PONG") {
		return ui.FormatTableRow(serviceID, address, colorizeNA("N/A"), colorizeStatus("BAD"), colorizeNA("Ping"), colorizeNA("N/A"), colorizeNA("N/A"), time.Now().Format("15:04:05"))
	}

	// Send INFO to get version
	version := "N/A"
	if _, err = writer.WriteString("INFO server\r\n"); err == nil {
		writer.Flush()
		infoSizeStr, err := reader.ReadString('\n') // Read bulk string header, e.g., "$1234"
		if err == nil && strings.HasPrefix(infoSizeStr, "$") {
			infoSize, _ := Atoi(strings.TrimSpace(infoSizeStr[1:]))
			if infoSize > 0 {
				buf := make([]byte, infoSize+2) // +2 for \r\n
				if _, err := io.ReadFull(reader, buf); err == nil {
					infoStr := string(buf)
					version = parseRedisInfo(infoStr, "redis_version")
					if version == "N/A" {
						version = parseRedisInfo(infoStr, "valkey_version")
					}
				}
			}
		}
	}

	return ui.FormatTableRow(serviceID, address, colorizeNA(ui.Truncate(version, 12)), colorizeStatus("OK"), colorizeNA("N/A"), colorizeNA("N/A"), colorizeNA("N/A"), time.Now().Format("15:04:05"))
}

// Atoi converts string to int, returns 0 on error
func Atoi(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return i
}

// parseRedisInfo extracts a specific key from the INFO command response.
func parseRedisInfo(info, key string) string {
	lines := strings.Split(info, "\r\n")
	for _, line := range lines {
		if strings.HasPrefix(line, key+":") {
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				return parts[1]
			}
		}
	}
	return "N/A"
}

// checkHTTPStatus checks a service via its HTTP /status endpoint
func checkHTTPStatus(service config.ServiceDefinition, serviceID, address string) ui.TableRow {
	statusURL := service.GetHTTP("/status")
	resp, err := http.Get(statusURL)
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
		colorizeNA(uptime),
		colorizeNA(goroutines),
		colorizeNA(mem),
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
