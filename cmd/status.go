package cmd

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os/exec"
	"regexp"
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

	var rows []ui.TableRow
	allServices := config.GetAllServices()

	if serviceShortName == "all" || serviceShortName == "" {
		for _, serviceDef := range allServices {
			row := checkServiceStatus(serviceDef)
			rows = append(rows, row)
			log(fmt.Sprintf("Service: %s, Type: %s, Status: %s", serviceDef.ID, serviceDef.Type, row[3]))
		}
	} else {
		def, err := config.Resolve(serviceShortName)
		if err != nil {
			return fmt.Errorf("failed to resolve service '%s': %w", serviceShortName, err)
		}

		row := checkServiceStatus(*def)
		rows = append(rows, row)
		log(fmt.Sprintf("Service: %s, Type: %s, Status: %s", def.ID, def.Type, row[3]))
	}

	// Render table
	table := ui.CreateServiceTable(rows)
	table.Render()

	return nil
}

// checkServiceStatus acts as a dispatcher, routing to the correct status checker based on service type.
func checkServiceStatus(service config.ServiceDefinition) ui.TableRow {
	// Define max lengths for columns
	const (
		maxServiceLen = 19
		maxAddressLen = 17
		maxVersionLen = 12
		maxUptimeLen  = 10
	)

	serviceID := ui.Truncate(service.ID, maxServiceLen)
	address := ui.Truncate(service.GetHost(), maxAddressLen)

	switch service.Type {
	case "cli":
		return checkCLIStatus(service, serviceID)
	case "os":
		return checkCacheStatus(service, serviceID, address)
	default:
		// All other types (cs, be, th) are HTTP services
		return checkHTTPStatus(service, serviceID, address)
	}
}

// colorizeNA colors "N/A" values dark gray, and leaves other values as-is.
func colorizeNA(value string) string {
	if value == "N/A" {
		return ui.Colorize(value, ui.ColorDarkGray)
	}
	return value
}

// checkCLIStatus checks if a CLI tool is installed and working
func checkCLIStatus(service config.ServiceDefinition, serviceID string) ui.TableRow {
	cmd := exec.Command("dex", "version")
	output, err := cmd.CombinedOutput()

	status := "OK"
	if err != nil {
		status = "BAD"
	}

	// Parse version: v0.3.0.main.5241102.2025-11-03-20-20-30... -> 0.3.0
	version := "N/A"
	outputStr := strings.TrimSpace(string(output))
	if strings.HasPrefix(outputStr, "v") {
		parts := strings.Split(outputStr, ".")
		if len(parts) >= 3 {
			version = strings.Join(parts[0:3], ".")[1:] // [1:] to remove 'v'
		}
	}

	return ui.FormatTableRow(
		serviceID,
		colorizeNA("N/A"), // Address is N/A for CLI
		colorizeNA(ui.Truncate(version, 12)),
		colorizeStatus(status),
		colorizeNA("N/A"),
		colorizeNA("N/A"),
		colorizeNA("N/A"),
		time.Now.Format("15:04:05"),
	)
}

// isCloudDomain checks if the domain is a known cloud Redis provider requiring TLS.
func isCloudDomain(domain string) bool {
	return strings.Contains(domain, "redis-cloud.com") || strings.Contains(domain, "redns.redis-cloud.com")
}

// checkCacheStatus checks a cache/db service (Redis/Valkey) with an optional AUTH and a PING command.
func checkCacheStatus(service config.ServiceDefinition, serviceID, address string) ui.TableRow {
	var conn net.Conn
	var err error

	// Set a 2-second timeout for the initial connection
	dialer := &net.Dialer{Timeout: 2 * time.Second}

	if isCloudDomain(service.Domain) {
		// --- FIX 1: Use TLS for cloud domains ---
		conn, err = tls.DialWithDialer(dialer, "tcp", service.GetHost(), &tls.Config{})
	} else {
		// Use plain TCP for local domains
		conn, err = dialer.Dial("tcp", service.GetHost())
	}

	if err != nil {
		return ui.FormatTableRow(serviceID, address, colorizeNA("N/A"), colorizeStatus("BAD"), colorizeNA("N/A"), colorizeNA("N/A"), colorizeNA("N/A"), time.Now().Format("15:04:05"))
	}
	defer func() { _ = conn.Close() }()

	// Set a deadline for all subsequent Read/Write operations
	if err := conn.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
		return ui.FormatTableRow(serviceID, address, colorizeNA("N/A"), colorizeStatus("BAD"), colorizeNA("N/A"), colorizeNA("N/A"), colorizeNA("N/A"), time.Now().Format("15:04:05"))
	}

	reader := bufio.NewReader(conn)

	// 1. Authenticate if password is provided
	if service.Credentials != nil && service.Credentials.Password != "" {
		// --- FIX 2: Use Username and Password for AUTH ---
		authCmd := fmt.Sprintf("AUTH %s %s\r\n", service.Credentials.Username, service.Credentials.Password)

		if _, err = conn.Write([]byte(authCmd)); err != nil {
			return ui.FormatTableRow(serviceID, address, colorizeNA("N/A"), colorizeStatus("BAD"), colorizeNA("Auth"), colorizeNA("N/A"), colorizeNA("N/A"), time.Now().Format("15:04:05"))
		}
		response, err := reader.ReadString('\n')
		if err != nil || !strings.HasPrefix(response, "+OK") {
			return ui.FormatTableRow(serviceID, address, colorizeNA("N/A"), colorizeStatus("BAD"), colorizeNA("Auth"), colorizeNA("N/A"), colorizeNA("N/A"), time.Now().Format("15:04:05"))
		}
		// Reset deadline for the next operation
		if err := conn.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
			return ui.FormatTableRow(serviceID, address, colorizeNA("N/A"), colorizeStatus("BAD"), colorizeNA("N/A"), colorizeNA("N/A"), colorizeNA("N/A"), time.Now().Format("15:04:05"))
		}
	}

	// 2. Ping
	if _, err = conn.Write([]byte("PING\r\n")); err != nil {
		return ui.FormatTableRow(serviceID, address, colorizeNA("N/A"), colorizeStatus("BAD"), colorizeNA("Ping"), colorizeNA("N/A"), colorizeNA("N/A"), time.Now().Format("15:04:05"))
	}
	response, err := reader.ReadString('\n')
	if err != nil || !strings.HasPrefix(response, "+PONG") {
		return ui.FormatTableRow(serviceID, address, colorizeNA("N/A"), colorizeStatus("BAD"), colorizeNA("Ping"), colorizeNA("N/A"), colorizeNA("N/A"), time.Now().Format("15:04:05"))
	}

	// 3. Get Version
	version := "N/A"
	// Reset deadline for the next operation
	if err := conn.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
		return ui.FormatTableRow(serviceID, address, colorizeNA("N/A"), colorizeStatus("OK"), colorizeNA("N/A"), colorizeNA("N/A"), colorizeNA("N/A"), time.Now().Format("15:04:05"))
	}

	if _, err = conn.Write([]byte("INFO server\r\n")); err == nil {
		// Read the bulk string response
		response, err = reader.ReadString('\n')
		if err == nil && strings.HasPrefix(response, "$") {
			// Read the info string itself
			infoData, _ := io.ReadAll(io.LimitReader(reader, 4096))
			infoStr := string(infoData)

			// Try to find redis_version or valkey_version
			re := regexp.MustCompile(`(redis_version|valkey_version):([0-9]+\.[0-9]+\.[0-9]+)`)
			matches := re.FindStringSubmatch(infoStr)
			if len(matches) >= 3 {
				version = matches[2]
			}
		}
	}

	return ui.FormatTableRow(serviceID, address, colorizeNA(ui.Truncate(version, 12)), colorizeStatus("OK"), colorizeNA("N/A"), colorizeNA("N/A"), colorizeNA("N/A"), time.Now().Format("15:04:05"))
}

// checkHTTPStatus checks a service via its HTTP /status endpoint
func checkHTTPStatus(service config.ServiceDefinition, serviceID, address string) ui.TableRow {
	statusURL := service.GetHTTP("/status")

	// Use a custom HTTP client with a 2-second timeout
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
