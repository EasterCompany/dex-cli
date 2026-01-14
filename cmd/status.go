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
	if serviceShortName == "--help" || serviceShortName == "-h" {
		ui.PrintHeader("Status Command Help")
		ui.PrintInfo("Usage: dex status [service|all]")
		fmt.Println()
		ui.PrintInfo("Description:")
		ui.PrintInfo("  Check the status of CLI and services.")
		ui.PrintInfo("  If no argument or 'all' is provided, checks all services.")
		ui.PrintInfo("  Otherwise, checks the specified service.")
		return nil
	}

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
// All handlers must return exactly 9 columns: SERVICE, ADDRESS, VERSION, BRANCH, COMMIT, STATUS, UPTIME, CPU, MEM.
func checkServiceStatus(service config.ServiceDefinition) ui.TableRow {
	serviceID := ui.Truncate(service.ShortName, maxServiceLen)
	address := ui.Truncate(service.GetHost(), maxAddressLen)

	switch service.Type {
	case "cli":
		return checkCLIStatus(service, serviceID, address)
	case "prd":
		return checkProdStatus(service, serviceID, address)
	case "os":
		// Specialized handling for Ollama
		if strings.Contains(strings.ToLower(service.ID), "ollama") || strings.Contains(strings.ToLower(service.ShortName), "ollama") {
			return checkOllamaStatus(service, serviceID, address)
		}
		// Specialized handling for Upstash (REST API)
		if strings.Contains(strings.ToLower(service.Domain), "upstash.io") {
			return checkUpstashStatus(service, serviceID, address)
		}
		return checkCacheStatus(service, serviceID, address)
	default: // All other service types are assumed to be HTTP-based (fe, be, cs, th)
		return checkHTTPStatus(service, serviceID, address)
	}
}

// checkUpstashStatus checks an Upstash service via its HTTP REST API.
func checkUpstashStatus(service config.ServiceDefinition, serviceID, address string) ui.TableRow {
	badStatusRow := func(reason string) ui.TableRow {
		// Log the failure reason for debugging
		logFile, _ := config.LogFile()
		if logFile != nil {
			_, _ = fmt.Fprintf(logFile, "[%s] Upstash check failed: %s\n", serviceID, reason)
		}
		return []string{
			serviceID,
			address,
			colorizeNA("Cloud"),   // VERSION
			colorizeNA("--"),      // BRANCH
			colorizeNA("--"),      // COMMIT
			colorizeStatus("BAD"), // STATUS
			colorizeNA("∞"),       // UPTIME
			colorizeNA("--"),      // CPU
			colorizeNA("--"),      // MEM
		}
	}

	url := fmt.Sprintf("https://%s/ping", service.Domain)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return badStatusRow(fmt.Sprintf("failed to create request: %v", err))
	}

	if service.Credentials != nil && service.Credentials.Password != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", service.Credentials.Password))
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return badStatusRow(fmt.Sprintf("request failed: %v", err))
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return badStatusRow(fmt.Sprintf("HTTP %d", resp.StatusCode))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return badStatusRow(fmt.Sprintf("failed to read body: %v", err))
	}

	var result struct {
		Result string `json:"result"`
		Error  string `json:"error"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return badStatusRow(fmt.Sprintf("failed to parse JSON: %v", err))
	}

	if result.Error != "" {
		return badStatusRow(result.Error)
	}

	if result.Result != "PONG" {
		return badStatusRow(fmt.Sprintf("unexpected result: %s", result.Result))
	}

	return []string{
		serviceID,
		address,
		colorizeNA("Cloud"), // VERSION
		colorizeNA("--"),    // BRANCH
		colorizeNA("--"),    // COMMIT
		colorizeStatus("OK"),
		colorizeNA("∞"),
		colorizeNA("--"),
		colorizeNA("--"),
	}
}

// checkProdStatus checks a production service via a simple HTTPS ping (ignoring port).
func checkProdStatus(service config.ServiceDefinition, serviceID, address string) ui.TableRow {
	badStatusRow := func(reason string) ui.TableRow {
		// Log the failure reason for debugging
		logFile, _ := config.LogFile()
		if logFile != nil {
			_, _ = fmt.Fprintf(logFile, "[%s] Production check failed: %s\n", serviceID, reason)
		}
		return []string{
			serviceID,
			address,
			colorizeNA("Live"),    // VERSION
			colorizeNA("--"),      // BRANCH
			colorizeNA("--"),      // COMMIT
			colorizeStatus("BAD"), // STATUS
			colorizeNA("--"),      // UPTIME
			colorizeNA("--"),      // CPU
			colorizeNA("--"),      // MEM
		}
	}

	// Use https://<domain> only, ignoring port for production sites
	url := fmt.Sprintf("https://%s", service.Domain)
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return badStatusRow(fmt.Sprintf("failed to reach production site: %v", err))
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return badStatusRow(fmt.Sprintf("HTTP %d", resp.StatusCode))
	}

	return []string{
		serviceID,
		address,
		colorizeNA("Live"), // VERSION
		colorizeNA("--"),   // BRANCH
		colorizeNA("--"),   // COMMIT
		colorizeStatus("OK"),
		colorizeNA("∞"),
		colorizeNA("--"),
		colorizeNA("--"),
	}
}

// colorizeNA colors "N/A" values dark gray, and leaves other values as-is.
func colorizeNA(value string) string {
	if value == "N/A" {
		value = "--"
	}
	if value == "--" || value == "" {
		return fmt.Sprintf("%s%s%s", ui.ColorDarkGray, value, ui.ColorReset)
	}
	return value
}

// formatUptime converts seconds into a human-readable uptime string
func formatUptime(seconds int) string {
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}

	days := seconds / 86400
	hours := (seconds % 86400) / 3600
	minutes := (seconds % 3600) / 60

	if days > 0 {
		return fmt.Sprintf("%dd%dh", days, hours)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh%dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}

// checkCLIStatus checks if the CLI tool is installed and working
// Returns 9 columns: SERVICE, ADDRESS, VERSION, BRANCH, COMMIT, STATUS, UPTIME, CPU, MEM
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
		colorizeNA("N/A"), // UPTIME
		colorizeNA("N/A"), // CPU
		colorizeNA("N/A"), // MEM
	}
}

// isCloudDomain checks if the domain is a known cloud Redis provider requiring TLS.
func isCloudDomain(domain string) bool {
	return strings.Contains(domain, "redis-cloud.com") ||
		strings.Contains(domain, "redns.redis-cloud.com") ||
		strings.Contains(domain, "upstash.io")
}

// isLocalAddress checks if the address is a local/localhost address
func isLocalAddress(domain string) bool {
	return domain == "localhost" ||
		domain == "127.0.0.1" ||
		strings.HasPrefix(domain, "127.") ||
		domain == "0.0.0.0" ||
		strings.HasPrefix(domain, "192.168.") ||
		strings.HasPrefix(domain, "10.") ||
		strings.HasPrefix(domain, "172.16.") ||
		strings.HasPrefix(domain, "172.17.") ||
		strings.HasPrefix(domain, "172.18.") ||
		strings.HasPrefix(domain, "172.19.") ||
		strings.HasPrefix(domain, "172.20.") ||
		strings.HasPrefix(domain, "172.21.") ||
		strings.HasPrefix(domain, "172.22.") ||
		strings.HasPrefix(domain, "172.23.") ||
		strings.HasPrefix(domain, "172.24.") ||
		strings.HasPrefix(domain, "172.25.") ||
		strings.HasPrefix(domain, "172.26.") ||
		strings.HasPrefix(domain, "172.27.") ||
		strings.HasPrefix(domain, "172.28.") ||
		strings.HasPrefix(domain, "172.29.") ||
		strings.HasPrefix(domain, "172.30.") ||
		strings.HasPrefix(domain, "172.31.")
}

// getSystemdServiceUptime gets the uptime of a systemd service
func getSystemdServiceUptime(serviceName string) string {
	cmd := exec.Command("systemctl", "show", serviceName, "--property=ActiveEnterTimestamp")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "N/A"
	}

	// Parse output like: ActiveEnterTimestamp=Mon 2025-11-10 16:52:38 GMT
	line := strings.TrimSpace(string(output))
	if !strings.HasPrefix(line, "ActiveEnterTimestamp=") {
		return "N/A"
	}

	timestampStr := strings.TrimPrefix(line, "ActiveEnterTimestamp=")
	if timestampStr == "" {
		return "N/A"
	}

	// Parse the timestamp
	layout := "Mon 2006-01-02 15:04:05 MST"
	startTime, err := time.Parse(layout, timestampStr)
	if err != nil {
		return "N/A"
	}

	// Calculate uptime in seconds
	uptimeSeconds := int(time.Since(startTime).Seconds())
	return formatUptime(uptimeSeconds)
}

// getSystemdServiceMemory gets the memory usage percentage of a systemd service
func getSystemdServiceMemory(serviceName string) string {
	cmd := exec.Command("systemctl", "show", serviceName, "--property=MemoryCurrent")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "N/A"
	}

	line := strings.TrimSpace(string(output))
	if !strings.HasPrefix(line, "MemoryCurrent=") {
		return "N/A"
	}

	memoryStr := strings.TrimPrefix(line, "MemoryCurrent=")
	var memoryBytes int64
	_, err = fmt.Sscanf(memoryStr, "%d", &memoryBytes)
	if err != nil || memoryBytes <= 0 {
		return "N/A"
	}

	// Get total system memory
	totalMemory, err := getTotalSystemMemory()
	if err != nil {
		return "N/A"
	}

	// Calculate percentage
	percentage := (float64(memoryBytes) / float64(totalMemory)) * 100
	return fmt.Sprintf("%.1f%%", percentage)
}

// getSystemdServiceCPU gets the average CPU usage of a systemd service
func getSystemdServiceCPU(serviceName string) string {
	// Get CPU usage and uptime
	cmd := exec.Command("systemctl", "show", serviceName, "--property=CPUUsageNSec", "--property=ActiveEnterTimestamp")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "N/A"
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var cpuNanoseconds int64
	var startTime time.Time

	for _, line := range lines {
		if strings.HasPrefix(line, "CPUUsageNSec=") {
			cpuStr := strings.TrimPrefix(line, "CPUUsageNSec=")
			_, _ = fmt.Sscanf(cpuStr, "%d", &cpuNanoseconds)
		} else if strings.HasPrefix(line, "ActiveEnterTimestamp=") {
			timestampStr := strings.TrimPrefix(line, "ActiveEnterTimestamp=")
			if timestampStr != "" {
				layout := "Mon 2006-01-02 15:04:05 MST"
				startTime, _ = time.Parse(layout, timestampStr)
			}
		}
	}

	if cpuNanoseconds <= 0 || startTime.IsZero() {
		return "N/A"
	}

	// Calculate elapsed time in nanoseconds
	elapsedNanoseconds := time.Since(startTime).Nanoseconds()
	if elapsedNanoseconds <= 0 {
		return "N/A"
	}

	// Calculate average CPU percentage
	// CPU time / elapsed time * 100 = percentage
	percentage := (float64(cpuNanoseconds) / float64(elapsedNanoseconds)) * 100
	return fmt.Sprintf("%.1f%%", percentage)
}

// getTotalSystemMemory reads the total system memory from /proc/meminfo
func getTotalSystemMemory() (int64, error) {
	cmd := exec.Command("grep", "MemTotal", "/proc/meminfo")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, err
	}

	// Parse output like: MemTotal:       131825740 kB
	line := strings.TrimSpace(string(output))
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return 0, fmt.Errorf("unexpected format")
	}

	var memoryKB int64
	_, err = fmt.Sscanf(fields[1], "%d", &memoryKB)
	if err != nil {
		return 0, err
	}

	// Convert KB to bytes
	return memoryKB * 1024, nil
}

// checkOllamaStatus checks an Ollama service via its HTTP API
// Returns 9 columns: SERVICE, ADDRESS, VERSION, BRANCH, COMMIT, STATUS, UPTIME, CPU, MEM
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
			colorizeNA("N/A"),     // CPU
			colorizeNA("N/A"),     // MEM
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

	// Get uptime, CPU, and memory from systemd if this is a local service
	uptime := "N/A"
	cpu := "N/A"
	mem := "N/A"
	if isLocalAddress(service.Domain) {
		uptime = getSystemdServiceUptime("ollama")
		cpu = getSystemdServiceCPU("ollama")
		mem = getSystemdServiceMemory("ollama")
	}

	// Successful status row
	return []string{
		serviceID,
		address,
		colorizeNA(ui.Truncate(versionData.Version, maxVersionLen)),
		colorizeNA("--"), // BRANCH
		colorizeNA("--"), // COMMIT
		colorizeStatus("OK"),
		colorizeNA(uptime),
		colorizeNA(cpu),
		colorizeNA(mem),
	}
}

// checkCacheStatus checks a cache/db service (Redis/Valkey) with a simplified PING command.
// Returns 9 columns: SERVICE, ADDRESS, VERSION, BRANCH, COMMIT, STATUS, UPTIME, CPU, MEM
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
			colorizeNA("N/A"),     // CPU
			colorizeNA("N/A"),     // MEM
		}
	}

	dialer := &net.Dialer{Timeout: 5 * time.Second} // Increased timeout for cloud instances
	host := service.GetHost()

	var conn net.Conn
	var err error
	useTLS := false

	// 1. Connection attempt with Protocol Intelligence
	if isCloudDomain(service.Domain) {
		useTLS = true
		tlsConfig := &tls.Config{
			ServerName: service.Domain, // Proper SNI for cloud Redis
		}
		conn, err = tls.DialWithDialer(dialer, "tcp", host, tlsConfig)

		// FALLBACK: If TLS handshake fails (port expects plain text), try PLAIN TCP
		if err != nil && (strings.Contains(err.Error(), "first record does not look like a TLS handshake") ||
			strings.Contains(err.Error(), "handshake failure")) {
			useTLS = false
			conn, err = dialer.Dial("tcp", host)
		}
	} else {
		// Otherwise, try plain TCP
		conn, err = dialer.Dial("tcp", host)
	}

	if err != nil {
		return badStatusRow(fmt.Sprintf("connection failed (TLS=%v): %v", useTLS, err))
	}
	defer func() { _ = conn.Close() }()

	if err := conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return badStatusRow(fmt.Sprintf("failed to set deadline: %v", err))
	}

	reader := bufio.NewReader(conn)

	// 2. Authenticate if password is provided
	if service.Credentials != nil && service.Credentials.Password != "" {
		var authCmd string
		if service.Credentials.Username != "" {
			authCmd = fmt.Sprintf("AUTH %s %s\r\n", service.Credentials.Username, service.Credentials.Password)
		} else {
			authCmd = fmt.Sprintf("AUTH %s\r\n", service.Credentials.Password)
		}

		if _, err = conn.Write([]byte(authCmd)); err != nil {
			return badStatusRow(fmt.Sprintf("failed to send AUTH command: %v", err))
		}

		response, err := reader.ReadString('\n')
		if err != nil {
			return badStatusRow(fmt.Sprintf("AUTH read error: %v", err))
		}

		if !strings.HasPrefix(response, "+OK") {
			// If we tried with username and failed, try one more time with JUST password as fallback
			if service.Credentials.Username != "" {
				// We need a fresh connection or to clear the buffer, but let's try a simple fallback first
				retryAuth := fmt.Sprintf("AUTH %s\r\n", service.Credentials.Password)
				if _, err = conn.Write([]byte(retryAuth)); err == nil {
					response, err = reader.ReadString('\n')
					if err == nil && strings.HasPrefix(response, "+OK") {
						goto authSuccess
					}
				}
			}
			return badStatusRow(fmt.Sprintf("AUTH failed: %s", strings.TrimSpace(response)))
		}
	}

authSuccess:
	// 3. Ping check
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

	// 3. Get Version and Uptime
	version := "N/A"
	uptime := "N/A"

	// Reset deadline for INFO/Version fetch
	if err := conn.SetDeadline(time.Now().Add(5 * time.Second)); err == nil {
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
				versionRe := regexp.MustCompile(`(redis_version|valkey_version):([0-9]+\.[0-9]+\.[0-9]+)`)
				versionMatches := versionRe.FindStringSubmatch(infoStr)
				if len(versionMatches) >= 3 && versionMatches[2] != "" {
					version = versionMatches[2]
				}

				// Find uptime_in_seconds and format it
				uptimeRe := regexp.MustCompile(`uptime_in_seconds:(\d+)`)
				uptimeMatches := uptimeRe.FindStringSubmatch(infoStr)
				if len(uptimeMatches) >= 2 {
					uptimeSeconds := 0
					_, _ = fmt.Sscanf(uptimeMatches[1], "%d", &uptimeSeconds)
					uptime = formatUptime(uptimeSeconds)
				}
			}
		}
	}

	// Get CPU and memory from systemd for local Redis services
	cpu := "N/A"
	mem := "N/A"
	if isLocalAddress(service.Domain) {
		// Try common Redis systemd service names
		cpu = getSystemdServiceCPU("redis")
		mem = getSystemdServiceMemory("redis")
		// If "redis" doesn't work, try "redis-server"
		if cpu == "N/A" {
			cpu = getSystemdServiceCPU("redis-server")
			mem = getSystemdServiceMemory("redis-server")
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
		colorizeNA(uptime),
		colorizeNA(cpu),
		colorizeNA(mem),
	}
}

// checkHTTPStatus checks a service via its new, unified /service endpoint
// Returns 9 columns: SERVICE, ADDRESS, VERSION, BRANCH, COMMIT, STATUS, UPTIME, CPU, MEM
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
		Metrics struct {
			CPU struct {
				Avg *float64 `json:"avg"`
			} `json:"cpu"`
			Memory struct {
				Avg *float64 `json:"avg"`
			} `json:"memory"`
		} `json:"metrics"`
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
			colorizeNA("N/A"),     // CPU
			colorizeNA("N/A"),     // MEM
		}
	}

	// Get the full JSON service report
	jsonResponse, err := utils.GetHTTPServiceReport(service)
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

	// If branch/commit are missing or "unknown", try to parse from Str
	if (branch == "" || branch == "unknown") || (commit == "" || commit == "unknown") {
		if parsed, err := git.Parse(report.Version.Str); err == nil {
			if parsed.Branch != "" {
				branch = parsed.Branch
			}
			if parsed.Commit != "" {
				commit = parsed.Commit
			}
		}
	}

	// Truncate commit hash if it's the full hash
	if len(commit) > 7 {
		commit = commit[:7]
	}

	// Use the parsed data for the table
	shortVersion := utils.ParseToShortVersion(report.Version.Str)

	// Cleanup Uptime for display
	uptime := ui.Truncate(report.Health.Uptime, maxUptimeLen)

	// Format CPU and Memory metrics
	cpu := "N/A"
	mem := "N/A"
	if report.Metrics.CPU.Avg != nil {
		cpu = fmt.Sprintf("%.1f%%", *report.Metrics.CPU.Avg)
	}
	if report.Metrics.Memory.Avg != nil {
		mem = fmt.Sprintf("%.1f MB", *report.Metrics.Memory.Avg)
	}

	return []string{
		serviceID,
		address,
		colorizeNA(ui.Truncate(shortVersion, maxVersionLen)),
		colorizeNA(ui.Truncate(branch, maxBranchLen)),
		colorizeNA(ui.Truncate(commit, maxCommitLen)),
		colorizeStatus(strings.ToUpper(report.Health.Status)),
		colorizeNA(uptime),
		colorizeNA(cpu),
		colorizeNA(mem),
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
