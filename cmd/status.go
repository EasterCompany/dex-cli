package cmd

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/git"
	"github.com/EasterCompany/dex-cli/ui"
	"github.com/EasterCompany/dex-cli/utils"
)

const (
	maxServiceLen = 8
	maxAddressLen = 16
	maxVersionLen = 8
	maxBranchLen  = 8
	maxCommitLen  = 8
	maxUptimeLen  = 16
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

	// Get the list of services *from the service-map.json*
	allServices, err := utils.GetConfiguredServices()
	if err != nil {
		return fmt.Errorf("failed to get configured services: %w", err)
	}

	var rows []ui.TableRow
	var servicesToCheck []config.ServiceDefinition

	if serviceShortName == "all" || serviceShortName == "" {
		servicesToCheck = allServices
	} else {
		// Find the specific service by its short name from the configured list
		var foundService *config.ServiceDefinition
		for i, s := range allServices {
			if s.ShortName == serviceShortName {
				// Use the address of the item in the slice
				foundService = &allServices[i]
				break
			}
		}

		if foundService == nil {
			return fmt.Errorf("service alias '%s' not found in configured services (service-map.json)", serviceShortName)
		}
		servicesToCheck = append(servicesToCheck, *foundService)
	}

	// Check status for all selected services
	for _, serviceDef := range servicesToCheck {
		row := checkServiceStatus(serviceDef)
		rows = append(rows, row)
		log(fmt.Sprintf("Service: %s, Type: %s, Address: %s, Status: %s", serviceDef.ID, serviceDef.Type, serviceDef.GetHost(), row[3]))
	}

	// Render table
	table := ui.CreateServiceTable(rows)
	table.Render()

	return nil
}

// checkServiceStatus acts as a dispatcher, routing to the correct status checker based on service type.
func checkServiceStatus(service config.ServiceDefinition) ui.TableRow {
	serviceID := ui.Truncate(service.ShortName, maxServiceLen)
	address := ui.Truncate(service.GetHost(), maxAddressLen)

	switch service.Type {
	case "cli":
		return checkCLIStatus(service, serviceID)
	case "os":
		return checkCacheStatus(service, serviceID, address)
	default:
		return checkHTTPStatus(service, serviceID, address)
	}
}

// colorizeNA colors "N/A" values dark gray, and leaves other values as-is.
func colorizeNA(value string) string {
	if value == "N/A" {
		return fmt.Sprintf("%s%s%s", ui.ColorDarkGray, value, ui.ColorReset)
	}
	return value
}

// checkCLIStatus checks if the CLI tool is installed and working
func checkCLIStatus(_ config.ServiceDefinition, serviceID string) ui.TableRow {
	cmd := exec.Command("dex", "version")
	output, err := cmd.CombinedOutput()

	status := "OK"
	if err != nil {
		status = "BAD"
	}

	version := "N/A"
	branch := "N/A"
	commit := "N/A"

	parsedVersion, err := git.Parse(strings.TrimSpace(string(output)))
	if err == nil {
		version = parsedVersion.Short()
		branch = parsedVersion.Branch
		commit = parsedVersion.Commit
	}

	return []string{
		serviceID,
		colorizeNA("N/A"),
		colorizeNA(ui.Truncate(version, maxVersionLen)),
		colorizeNA(branch),
		colorizeNA(commit),
		colorizeStatus(status),
		colorizeNA("N/A"),
		colorizeNA("N/A"),
		colorizeNA("N/A"),
		time.Now().Format("15:04:05"),
	}
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
	host := service.GetHost() // Get "domain:port" from the definition

	if isCloudDomain(service.Domain) {
		// --- FIX 1: Use TLS for cloud domains ---
		conn, err = tls.DialWithDialer(dialer, "tcp", host, &tls.Config{})
	} else {
		// Use plain TCP for local domains
		conn, err = dialer.Dial("tcp", host)
	}

	if err != nil {
		return []string{serviceID, address, colorizeNA("N/A"), colorizeStatus("BAD"), colorizeNA("N/A"), colorizeNA("N/A"), colorizeNA("N/A"), time.Now().Format("15:04:05")}
	}
	defer func() { _ = conn.Close() }()

	// Set a deadline for all subsequent Read/Write operations
	if err := conn.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
		return []string{serviceID, address, colorizeNA("N/A"), colorizeStatus("BAD"), colorizeNA("N/A"), colorizeNA("N/A"), colorizeNA("N/A"), time.Now().Format("15:04:05")}
	}

	reader := bufio.NewReader(conn)

	// 1. Authenticate if password is provided
	if service.Credentials != nil && service.Credentials.Password != "" {
		// --- FIX 2: Use Username and Password for AUTH ---
		var authCmd string
		if service.Credentials.Username != "" && service.Credentials.Username != "default" {
			authCmd = fmt.Sprintf("AUTH %s %s\r\n", service.Credentials.Username, service.Credentials.Password)
		} else {
			// Fallback for older Redis (just password) or "default" user
			authCmd = fmt.Sprintf("AUTH %s\r\n", service.Credentials.Password)
		}

		if _, err = conn.Write([]byte(authCmd)); err != nil {
			return []string{serviceID, address, colorizeNA("N/A"), colorizeStatus("BAD"), colorizeNA("Auth"), colorizeNA("N/A"), colorizeNA("N/A"), time.Now().Format("15:04:05")}
		}
		response, err := reader.ReadString('\n')
		if err != nil || !strings.HasPrefix(response, "+OK") {
			// Try fallback AUTH (just password) if user+pass failed
			if strings.Contains(authCmd, " ") {
				authCmd = fmt.Sprintf("AUTH %s\r\n", service.Credentials.Password)
				if _, err = conn.Write([]byte(authCmd)); err == nil {
					response, err = reader.ReadString('\n')
				}
			}

			// If it still fails
			if err != nil || !strings.HasPrefix(response, "+OK") {
				return []string{serviceID, address, colorizeNA("N/A"), colorizeStatus("BAD"), colorizeNA("Auth"), colorizeNA("N/A"), colorizeNA("N/A"), time.Now().Format("15:04:05")}
			}
		}
		// Reset deadline for the next operation
		if err := conn.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
			return []string{serviceID, address, colorizeNA("N/A"), colorizeStatus("BAD"), colorizeNA("N/A"), colorizeNA("N/A"), colorizeNA("N/A"), time.Now().Format("15:04:05")}
		}
	}

	// 2. Ping
	if _, err = conn.Write([]byte("PING\r\n")); err != nil {
		return []string{serviceID, address, colorizeNA("N/A"), colorizeStatus("BAD"), colorizeNA("Ping"), colorizeNA("N/A"), colorizeNA("N/A"), time.Now().Format("15:04:05")}
	}
	response, err := reader.ReadString('\n')
	if err != nil || !strings.HasPrefix(response, "+PONG") {
		return []string{serviceID, address, colorizeNA("N/A"), colorizeStatus("BAD"), colorizeNA("Ping"), colorizeNA("N/A"), colorizeNA("N/A"), time.Now().Format("15:04:05")}
	}

	// 3. Get Version
	version := "N/A"
	// Reset deadline for the next operation
	if err := conn.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
		return []string{serviceID, address, colorizeNA("N/A"), colorizeStatus("OK"), colorizeNA("N/A"), colorizeNA("N/A"), colorizeNA("N/A"), time.Now().Format("15:04:05")}
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
			if len(matches) >= 3 && matches[2] != "" {
				version = matches[2]
			} else {
				version = "N/A"
			}
		}
	}

	return []string{serviceID, address, colorizeNA(ui.Truncate(version, maxVersionLen)), colorizeStatus("OK"), colorizeNA("N/A"), colorizeNA("N/A"), colorizeNA("N/A"), time.Now().Format("15:04:05")}
}

// checkHTTPStatus checks a service via its new, unified /service endpoint
func checkHTTPStatus(service config.ServiceDefinition, serviceID, address string) ui.TableRow {
	// A minimal struct to unmarshal only what we need for the status table
	type serviceReport struct {
		Version struct {
			Str string `json:"str"`
		} `json:"version"`
		Health struct {
			Status string `json:"status"`
			Uptime string `json:"uptime"`
		} `json:"health"`
	}

	// Get the raw JSON response from the service
	jsonResponse, err := utils.GetHTTPVersion(service)
	if err != nil {
		return []string{
			serviceID,
			address,
			colorizeNA("N/A"),
			colorizeStatus("BAD"),
			colorizeNA("N/A"),
			colorizeNA("N/A"),
			colorizeNA("N/A"),
			time.Now().Format("15:04:05"),
		}
	}

	var report serviceReport
	if err := json.Unmarshal([]byte(jsonResponse), &report); err != nil {
		// If parsing fails, the service might be returning a non-JSON error
		return []string{
			serviceID,
			address,
			colorizeNA("N/A"),
			colorizeStatus("BAD"),
			colorizeNA("N/A"),
			colorizeNA("N/A"),
			colorizeNA("N/A"),
			time.Now().Format("15:04:05"),
		}
	}

	// Use the parsed data for the table
	shortVersion := utils.ParseToShortVersion(report.Version.Str)
	return []string{
		serviceID,
		address,
		colorizeNA(ui.Truncate(shortVersion, maxVersionLen)),
		colorizeStatus(strings.ToUpper(report.Health.Status)),
		colorizeNA(ui.Truncate(report.Health.Uptime, 10)),
		colorizeNA("N/A"), // Placeholder for future metrics
		colorizeNA("N/A"), // Placeholder for future metrics
		time.Now().Format("15:04:05"),
	}
}

// colorizeStatus applies color coding to the status string.
func colorizeStatus(status string) string {
	switch status {
	case "OK", "HEALTHY":
		return fmt.Sprintf("%s%s%s", ui.ColorGreen, status, ui.ColorReset)
	case "BAD":
		return fmt.Sprintf("%s%s%s", ui.ColorBrightRed, status, ui.ColorReset)
	case "N/A":
		return fmt.Sprintf("%s%s%s", ui.ColorDarkGray, status, ui.ColorReset)
	default:
		return status
	}
}
