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
	maxSourceLen  = 8
	maxStatusLen  = 8
	maxUptimeLen  = 16
)

const (
	CheckMark = "✅"
	CrossMark = "❌"
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
		// NOTE: Status column is at index 5.
		log(fmt.Sprintf("Service: %s, Type: %s, Address: %s, Status: %s", serviceDef.ID, serviceDef.Type, serviceDef.GetHost(), row[5]))
	}

	// Render table
	table := ui.CreateServiceTable(rows)
	table.Render()

	return nil
}

// checkServiceStatus acts as a dispatcher, routing to the correct status checker based on service type.
// All handlers must return exactly 8 columns: SERVICE, ADDRESS, VERSION, BRANCH, COMMIT, STATUS, UPTIME, SOURCE.
func checkServiceStatus(service config.ServiceDefinition) ui.TableRow {
	serviceID := ui.Truncate(service.ShortName, maxServiceLen)
	address := ui.Truncate(service.GetHost(), maxAddressLen)

	switch service.Type {
	case "cli":
		return checkCLIStatus(service, serviceID, address)
	case "os":
		// Special handling for ollama (check by ID or short name)
		if strings.Contains(strings.ToLower(service.ID), "ollama") || strings.Contains(strings.ToLower(service.ShortName), "ollama") {
			return checkOllamaStatus(service, serviceID, address)
		}
		return checkCacheStatus(service, serviceID, address)
	default: // All other service types are assumed to be HTTP-based (fe, be, cs, th)
		return checkHTTPStatus(service, serviceID, address)
	}
}

// colorizeNA colors "N/A" values dark gray, and leaves other values as-is.
func colorizeNA(value string) string {
	if value == "N/A" || value == "--" || value == "" {
		return fmt.Sprintf("%s%s%s", ui.ColorDarkGray, value, ui.ColorReset)
	}
	return value
}

// checkCLIStatus checks if the CLI tool is installed and working
// Returns 8 columns: SERVICE, ADDRESS, VERSION, BRANCH, COMMIT, STATUS, UPTIME, SOURCE
func checkCLIStatus(service config.ServiceDefinition, serviceID, address string) ui.TableRow {
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

	// Ensure the ADDRESS column is N/A for a local CLI, even if host is set.
	if address == "" {
		address = "N/A"
	}

	return []string{
		serviceID,
		colorizeNA(address), // ADDRESS (Likely "N/A")
		colorizeNA(ui.Truncate(version, maxVersionLen)),
		colorizeNA(branch),
		colorizeNA(commit),
		colorizeStatus(status),
		colorizeNA("N/A"),
	}
}

// isCloudDomain checks if the domain is a known cloud Redis provider requiring TLS.
func isCloudDomain(domain string) bool {
	return strings.Contains(domain, "redis-cloud.com") || strings.Contains(domain, "redns.redis-cloud.com")
}

// checkOllamaStatus checks an Ollama service via its HTTP API
// Returns 8 columns: SERVICE, ADDRESS, VERSION, BRANCH, COMMIT, STATUS, UPTIME, SOURCE
func checkOllamaStatus(service config.ServiceDefinition, serviceID, address string) ui.TableRow {
	badStatusRow := func() ui.TableRow {
		return []string{
			serviceID,
			address,
			colorizeNA("N/A"),     // VERSION
			colorizeNA("--"),      // BRANCH
			colorizeNA("--"),      // COMMIT
			colorizeStatus("BAD"), // STATUS
			colorizeNA("N/A"),     // UPTIME
		}
	}

	// Build the ollama version endpoint URL
	host := service.GetHost()
	url := fmt.Sprintf("http://%s/api/version", host)

	// Try to fetch version info
	resp, err := utils.FetchURL(url, 2*time.Second)
	if err != nil {
		return badStatusRow()
	}

	// Parse the JSON response
	var versionData struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal([]byte(resp), &versionData); err != nil {
		return badStatusRow()
	}

	// Successful status row
	return []string{
		serviceID,
		address,
		colorizeNA(ui.Truncate(versionData.Version, maxVersionLen)),
		colorizeNA("--"), // BRANCH
		colorizeNA("--"), // COMMIT
		colorizeStatus("OK"),
		colorizeNA("N/A"), // UPTIME
	}
}

// checkCacheStatus checks a cache/db service (Redis/Valkey) with a simplified PING command.
// Returns 8 columns: SERVICE, ADDRESS, VERSION, BRANCH, COMMIT, STATUS, UPTIME, SOURCE
func checkCacheStatus(service config.ServiceDefinition, serviceID, address string) ui.TableRow {
	badStatusRow := func(reason string) ui.TableRow {
		// Log the failure reason for debugging
		logFile, _ := config.LogFile()
		if logFile != nil {
			_, _ = fmt.Fprintf(logFile, "[%s] Cache check failed: %s\n", serviceID, reason)
		}
		return []string{
			serviceID,
			address,
			colorizeNA("N/A"),     // VERSION
			colorizeNA("--"),      // BRANCH
			colorizeNA("--"),      // COMMIT
			colorizeStatus("BAD"), // STATUS
			colorizeNA("N/A"),     // UPTIME
		}
	}

	dialer := &net.Dialer{Timeout: 5 * time.Second} // Increased timeout for cloud instances
	host := service.GetHost()

	var conn net.Conn
	var err error
	useTLS := false

	// Try plain TCP connection first
	conn, err = dialer.Dial("tcp", host)

	// If plain connection fails and this is a cloud domain, try TLS
	if err != nil && isCloudDomain(service.Domain) {
		useTLS = true
		tlsConfig := &tls.Config{
			ServerName: service.Domain, // Proper SNI for cloud Redis
		}
		conn, err = tls.DialWithDialer(dialer, "tcp", host, tlsConfig)
	}

	if err != nil {
		return badStatusRow(fmt.Sprintf("connection failed (TLS=%v): %v", useTLS, err))
	}
	defer func() { _ = conn.Close() }()

	if err := conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return badStatusRow(fmt.Sprintf("failed to set deadline: %v", err))
	}

	reader := bufio.NewReader(conn)

	// 1. Authenticate if password is provided
	if service.Credentials != nil && service.Credentials.Password != "" {
		// Try AUTH with username first (Redis 6+)
		authCmd := fmt.Sprintf("AUTH %s %s\r\n", service.Credentials.Username, service.Credentials.Password)
		if _, err = conn.Write([]byte(authCmd)); err != nil {
			return badStatusRow(fmt.Sprintf("failed to send AUTH command: %v", err))
		}
		response, err := reader.ReadString('\n')

		// If 2-arg AUTH failed, try simple password AUTH (Redis <6 or ACL not used)
		if err != nil || !strings.HasPrefix(response, "+OK") {
			simpleAuthCmd := fmt.Sprintf("AUTH %s\r\n", service.Credentials.Password)
			if _, err = conn.Write([]byte(simpleAuthCmd)); err != nil {
				return badStatusRow(fmt.Sprintf("failed to send simple AUTH: %v", err))
			}
			response, err = reader.ReadString('\n')
			if err != nil {
				return badStatusRow(fmt.Sprintf("AUTH read error: %v", err))
			}
			if !strings.HasPrefix(response, "+OK") {
				return badStatusRow(fmt.Sprintf("AUTH failed: %s", strings.TrimSpace(response)))
			}
		}
	}

	// 2. Ping check
	if _, err = conn.Write([]byte("PING\r\n")); err != nil {
		return badStatusRow(fmt.Sprintf("PING write failed: %v", err))
	}
	response, err := reader.ReadString('\n')
	if err != nil {
		return badStatusRow(fmt.Sprintf("PING read failed: %v", err))
	}
	if !strings.HasPrefix(response, "+PONG") {
		return badStatusRow(fmt.Sprintf("PING response invalid: %s", strings.TrimSpace(response)))
	}

	// 3. Get Version
	version := "N/A"

	// Reset deadline for INFO/Version fetch
	if err := conn.SetDeadline(time.Now().Add(2 * time.Second)); err == nil {
		if _, err = conn.Write([]byte("INFO server\r\n")); err == nil {
			responseHeader, err := reader.ReadString('\n')
			if err == nil && strings.HasPrefix(responseHeader, "$") {
				// Read only a limited amount to find the version, as the bulk string can be large
				sizeStr := strings.TrimSpace(responseHeader[1:])
				size, _ := fmt.Sscanf(sizeStr, "%d")
				limit := 4096
				if size > 0 && size < limit {
					limit = size
				}

				infoData, _ := io.ReadAll(io.LimitReader(reader, int64(limit)))
				infoStr := string(infoData)

				// Find redis_version or valkey_version
				re := regexp.MustCompile(`(redis_version|valkey_version):([0-9]+\.[0-9]+\.[0-9]+)`)
				matches := re.FindStringSubmatch(infoStr)
				if len(matches) >= 3 && matches[2] != "" {
					version = matches[2]
				}
			}
		}
	}

	// Successful status row
	return []string{
		serviceID,
		address,
		colorizeNA(ui.Truncate(version, maxVersionLen)),
		colorizeNA("--"), // BRANCH
		colorizeNA("--"), // COMMIT
		colorizeStatus("OK"),
		colorizeNA("N/A"), // UPTIME
	}
}

// checkHTTPStatus checks a service via its new, unified /service endpoint
// Returns 8 columns: SERVICE, ADDRESS, VERSION, BRANCH, COMMIT, STATUS, UPTIME, SOURCE
func checkHTTPStatus(service config.ServiceDefinition, serviceID, address string) ui.TableRow {
	// A struct to unmarshal the necessary fields for the status table
	type serviceReport struct {
		Version struct {
			Str string `json:"str"`
			Obj struct {
				Branch string `json:"branch"`
				Commit string `json:"commit"`
			} `json:"obj"`
		} `json:"version"`
		Health struct {
			Status string `json:"status"`
			Uptime string `json:"uptime"`
		} `json:"health"`
	}

	badStatusRow := func() ui.TableRow {
		return []string{
			serviceID,
			address,
			colorizeNA("N/A"),     // VERSION
			colorizeNA("--"),      // BRANCH
			colorizeNA("--"),      // COMMIT
			colorizeStatus("BAD"), // STATUS
			colorizeNA("N/A"),     // UPTIME
		}
	}

	// Get the raw JSON response from the service
	jsonResponse, err := utils.GetHTTPVersion(service)
	if err != nil {
		return badStatusRow()
	}

	var report serviceReport
	if err := json.Unmarshal([]byte(jsonResponse), &report); err != nil {
		// If parsing fails, return BAD status
		return badStatusRow()
	}

	// Extract remote version info
	branch := report.Version.Obj.Branch
	commit := report.Version.Obj.Commit

	// Use the parsed data for the table
	shortVersion := utils.ParseToShortVersion(report.Version.Str)

	// Cleanup Uptime for display
	uptime := ui.Truncate(report.Health.Uptime, maxUptimeLen)

	return []string{
		serviceID,
		address,
		colorizeNA(ui.Truncate(shortVersion, maxVersionLen)),
		colorizeNA(ui.Truncate(branch, maxBranchLen)),
		colorizeNA(ui.Truncate(commit, maxCommitLen)),
		colorizeStatus(strings.ToUpper(report.Health.Status)),
		colorizeNA(uptime),
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
