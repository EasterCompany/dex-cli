package cmd

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
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

// checkSourceExists checks if the local source directory exists.
func checkSourceExists(path string) bool {
	if path == "" {
		return false
	}
	// Use os.Stat to check if the path exists and is a directory
	info, err := os.Stat(path)
	if err != nil {
		// If error is os.IsNotExist, it doesn't exist. Other errors (like permission) should also fail the check.
		return false
	}
	return info.IsDir()
}

// getLocalGitInfo attempts to read the current branch and short commit hash from a local source directory.
// It returns the branch, commit, and a boolean indicating if the path exists AND git commands were successful.
func getLocalGitInfo(path string) (branch string, commit string, ok bool) {
	if !checkSourceExists(path) {
		return "", "", false
	}

	// 1. Get branch
	cmdBranch := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmdBranch.Dir = path
	branchBytes, err := cmdBranch.Output()
	if err != nil {
		// Directory exists, but git command failed (maybe not a git repo or other issue)
		return "", "", false
	}
	branch = strings.TrimSpace(string(branchBytes))

	// 2. Get short commit hash
	cmdCommit := exec.Command("git", "rev-parse", "--short", "HEAD")
	cmdCommit.Dir = path
	commitBytes, err := cmdCommit.Output()
	if err != nil {
		// Directory exists and branch found, but commit failed
		return branch, "", false
	}
	commit = strings.TrimSpace(string(commitBytes))

	return branch, commit, true
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

	// Determine SOURCE status (Only applies to CLI, FE, BE, CS, TH)
	sourceStatus := colorizeNA("N/A")

	if service.Source != "" {
		if !checkSourceExists(service.Source) {
			sourceStatus = CrossMark // Source provided but path does not exist
		} else {
			// Source path exists. Now check version match.
			localBranch, localCommit, gitFound := getLocalGitInfo(service.Source)

			if !gitFound || localBranch == "" || localCommit == "" {
				sourceStatus = CrossMark
			} else if localBranch == branch && localCommit == commit {
				sourceStatus = CheckMark // Match!
			} else {
				sourceStatus = CrossMark // Mismatch.
			}
		}
	}

	return []string{
		serviceID,
		colorizeNA(address), // ADDRESS (Likely "N/A")
		colorizeNA(ui.Truncate(version, maxVersionLen)),
		colorizeNA(branch),
		colorizeNA(commit),
		colorizeStatus(status),
		colorizeNA("N/A"), // UPTIME
		sourceStatus,      // NEW SOURCE COLUMN
	}
}

// isCloudDomain checks if the domain is a known cloud Redis provider requiring TLS.
func isCloudDomain(domain string) bool {
	return strings.Contains(domain, "redis-cloud.com") || strings.Contains(domain, "redns.redis-cloud.com")
}

// checkCacheStatus checks a cache/db service (Redis/Valkey) with a simplified PING command.
// Returns 8 columns: SERVICE, ADDRESS, VERSION, BRANCH, COMMIT, STATUS, UPTIME, SOURCE
func checkCacheStatus(service config.ServiceDefinition, serviceID, address string) ui.TableRow {
	// Source is N/A for OS/Cache services
	sourceStatus := CrossMark

	badStatusRow := func() ui.TableRow {
		return []string{
			serviceID,
			address,
			colorizeNA("N/A"),     // VERSION
			colorizeNA("--"),      // BRANCH
			colorizeNA("--"),      // COMMIT
			colorizeStatus("BAD"), // STATUS
			colorizeNA("N/A"),     // UPTIME
			sourceStatus,          // SOURCE (N/A)
		}
	}

	dialer := &net.Dialer{Timeout: 2 * time.Second}
	host := service.GetHost()

	var conn net.Conn
	var err error

	// Check if TLS is required (retained for redundancy against potential remote cloud domains)
	if isCloudDomain(service.Domain) {
		conn, err = tls.DialWithDialer(dialer, "tcp", host, &tls.Config{})
	} else {
		conn, err = dialer.Dial("tcp", host)
	}

	if err != nil {
		return badStatusRow()
	}
	defer func() { _ = conn.Close() }()

	if err := conn.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
		return badStatusRow()
	}

	reader := bufio.NewReader(conn)

	// 1. Authenticate if password is provided
	if service.Credentials != nil && service.Credentials.Password != "" {
		authCmd := fmt.Sprintf("AUTH %s %s\r\n", service.Credentials.Username, service.Credentials.Password)
		if _, err = conn.Write([]byte(authCmd)); err != nil {
			return badStatusRow()
		}
		response, err := reader.ReadString('\n')

		// If 2-factor AUTH failed or another error occurred, try simple password AUTH
		if err != nil || !strings.HasPrefix(response, "+OK") {
			simpleAuthCmd := fmt.Sprintf("AUTH %s\r\n", service.Credentials.Password)
			if _, err = conn.Write([]byte(simpleAuthCmd)); err != nil {
				return badStatusRow()
			}
			response, err = reader.ReadString('\n')
			if err != nil || !strings.HasPrefix(response, "+OK") {
				return badStatusRow()
			}
		}
	}

	// 2. Ping check
	if _, err = conn.Write([]byte("PING\r\n")); err != nil {
		return badStatusRow()
	}
	response, err := reader.ReadString('\n')
	if err != nil || !strings.HasPrefix(response, "+PONG") {
		return badStatusRow()
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
		sourceStatus,      // SOURCE (N/A)
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
			colorizeNA("N/A"),     // SOURCE (Default N/A if service check fails)
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

	// Determine SOURCE status (Only applies to CLI, FE, BE, CS, TH)
	sourceStatus := colorizeNA("N/A")
	if service.Source != "" {
		if !checkSourceExists(service.Source) {
			sourceStatus = CrossMark // Source provided but path does not exist
		} else {
			// Source path exists. Now check version match.
			localBranch, localCommit, gitFound := getLocalGitInfo(service.Source)

			if !gitFound || localBranch == "" || localCommit == "" {
				sourceStatus = CrossMark
			} else if localBranch == branch && localCommit == commit {
				sourceStatus = CheckMark // Match!
			} else {
				sourceStatus = CrossMark // Mismatch.
			}
		}
	}

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
		sourceStatus, // NEW SOURCE COLUMN
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
