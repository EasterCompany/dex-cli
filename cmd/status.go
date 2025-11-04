package cmd

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/git"
	"github.com/EasterCompany/dex-cli/health"
	"github.com/EasterCompany/dex-cli/ui"
)

// getConfiguredServices loads the service-map.json and merges its values
// with the master service definitions. This ensures user-configured
// domains, ports, and credentials are used.
func getConfiguredServices() ([]config.ServiceDefinition, error) {
	// 1. Load the user's service-map.json
	serviceMap, err := config.LoadServiceMapConfig()
	if err != nil {
		if os.IsNotExist(err) {
			// If it doesn't exist, use the default map
			serviceMap = config.DefaultServiceMapConfig()
		} else {
			// Other error
			return nil, fmt.Errorf("failed to load service-map.json: %w", err)
		}
	}

	// 2. Get the hardcoded master list of all *possible* services
	// This master list knows the ShortName, Type, ID, etc.
	masterList := config.GetAllServices()
	masterDefs := make(map[string]config.ServiceDefinition)
	for _, def := range masterList {
		masterDefs[def.ID] = def
	}

	// 3. Create the final list by merging the service map values
	var configuredServices []config.ServiceDefinition

	for serviceType, serviceEntries := range serviceMap.Services {
		for _, entry := range serviceEntries {
			// Find the master definition for this service ID
			masterDef, ok := masterDefs[entry.ID]
			if !ok {
				// This service is in service-map.json but not in the hardcoded master list
				// We can't check it if we don't know its type, shortname etc.
				// For now, we'll just use the entry data directly, but 'Type' and 'ShortName' might be incomplete.
				// A better approach: The masterDef *must* exist.
				// Let's assume for status, we only check services known to the CLI.
				continue
			}

			// Merge: Use master def as base, but override with user's config
			masterDef.Type = serviceType // Ensure type is from the map key
			if entry.Domain != "" {
				masterDef.Domain = entry.Domain
			}
			if entry.Port != "" {
				masterDef.Port = entry.Port
			}
			if entry.Credentials != nil {
				masterDef.Credentials = entry.Credentials
			}

			configuredServices = append(configuredServices, masterDef)
		}
	}

	// Sort by port to maintain a consistent order
	sort.Slice(configuredServices, func(i, j int) bool {
		return configuredServices[i].Port < configuredServices[j].Port
	})

	return configuredServices, nil
}

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
	allServices, err := getConfiguredServices()
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
	// Define max lengths for columns
	const (
		maxServiceLen = 19
		maxAddressLen = 17
		maxVersionLen = 12
		maxUptimeLen  = 10
	)

	// Use the ShortName from the definition for the table
	serviceID := ui.Truncate(service.ShortName, maxServiceLen)
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
		return fmt.Sprintf("%s%s%s", ui.ColorDarkGray, value, ui.ColorReset)
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

	version := "N/A"
	parsedVersion, err := git.Parse(strings.TrimSpace(string(output)))
	if err == nil {
		version = parsedVersion.Short()
	}

	return []string{
		serviceID,
		colorizeNA("N/A"), // Address is N/A for CLI
		colorizeNA(ui.Truncate(version, 12)),
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
			if len(matches) >= 3 {
				version = matches[2]
			}
		}
	}

	return []string{serviceID, address, colorizeNA(ui.Truncate(version, 12)), colorizeStatus("OK"), colorizeNA("N/A"), colorizeNA("N/A"), colorizeNA("N/A"), time.Now().Format("15:04:05")}
}

// checkHTTPStatus checks a service via its HTTP /status endpoint
func checkHTTPStatus(service config.ServiceDefinition, serviceID, address string) ui.TableRow {
	versionURL := service.GetHTTP("/version")
	statusURL := service.GetHTTP("/status")

	// Use a custom HTTP client with a 2-second timeout
	client := http.Client{
		Timeout: 2 * time.Second,
	}

	// Get Version
	version := "N/A"
	resp, err := client.Get(versionURL)
	if err == nil {
		defer func() { _ = resp.Body.Close() }()
		body, err := io.ReadAll(resp.Body)
		if err == nil {
			var versionResp map[string]string
			if err := json.Unmarshal(body, &versionResp); err == nil {
				parsedVersion, err := git.Parse(versionResp["version"])
				if err == nil {
					version = parsedVersion.Short()
				}
			}
		}
	}

	// Get Status
	resp, err = client.Get(statusURL)
	if err != nil {
		return []string{serviceID, address, colorizeNA(version), colorizeStatus("BAD"), colorizeNA("N/A"), colorizeNA("N/A"), colorizeNA("N/A"), time.Now().Format("15:04:05")}
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return []string{serviceID, address, colorizeNA(version), colorizeStatus("BAD"), colorizeNA("N/A"), colorizeNA("N/A"), colorizeNA("N/A"), time.Now().Format("15:04:05")}
	}

	var statusResp health.StatusResponse
	if err := json.Unmarshal(body, &statusResp); err != nil {
		return []string{serviceID, address, colorizeNA(version), colorizeStatus("BAD"), colorizeNA("N/A"), colorizeNA("N/A"), colorizeNA("N/A"), time.Now().Format("15:04:05")}
	}

	uptime := ui.Truncate(formatUptime(time.Duration(statusResp.Uptime)*time.Second), 10)

	goroutines := fmt.Sprintf("%d", statusResp.Metrics.Goroutines)
	mem := fmt.Sprintf("%.2f", statusResp.Metrics.MemoryAllocMB)

	return []string{
		serviceID,
		address,
		colorizeNA(ui.Truncate(version, 12)),
		colorizeStatus(statusResp.Status),
		colorizeNA(uptime),
		colorizeNA(goroutines),
		colorizeNA(mem),
		time.Unix(statusResp.Timestamp, 0).Format("15:00:00"), // Shortened timestamp
	}
}

// colorizeStatus applies color coding to the status string.
func colorizeStatus(status string) string {
	switch status {
	case "OK":
		return fmt.Sprintf("%s%s%s", ui.ColorGreen, status, ui.ColorReset)
	case "BAD":
		return fmt.Sprintf("%s%s%s", ui.ColorBrightRed, status, ui.ColorReset)
	case "N/A":
		return fmt.Sprintf("%s%s%s", ui.ColorDarkGray, status, ui.ColorReset)
	default:
		return status
	}
}
