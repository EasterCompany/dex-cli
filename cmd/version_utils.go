package cmd

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/EasterCompany/dex-cli/config"
)

// getServiceVersion fetches the version of a service based on its type.
func getServiceVersion(service config.ServiceDefinition) string {
	switch service.Type {
	case "cli":
		return getCLIVersion()
	case "os":
		return getCacheVersion(service)
	default:
		return getHTTPVersion(service)
	}
}

// getCLIVersion executes `dex version` to get the CLI's version.
func getCLIVersion() string {
	// Use the direct path to the newly built binary to ensure we get the new version.
	dexPath, err := config.ExpandPath("~/Dexter/bin/dex")
	if err != nil {
		return "N/A"
	}

	cmd := exec.Command(dexPath, "version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "N/A"
	}
	// The `version` command now outputs "v: <version>", so we return the full string.
	return strings.TrimSpace(string(output))
}

// getCacheVersion connects to a cache service to get its version.
func getCacheVersion(service config.ServiceDefinition) string {
	var conn net.Conn
	var err error

	dialer := &net.Dialer{Timeout: 2 * time.Second}
	host := service.GetHost()

	if isCloudDomain(service.Domain) {
		conn, err = tls.DialWithDialer(dialer, "tcp", host, &tls.Config{})
	} else {
		conn, err = dialer.Dial("tcp", host)
	}

	if err != nil {
		return "N/A"
	}
	defer func() { _ = conn.Close() }()

	if err := conn.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
		return "N/A"
	}

	reader := bufio.NewReader(conn)

	if service.Credentials != nil && service.Credentials.Password != "" {
		var authCmd string
		if service.Credentials.Username != "" && service.Credentials.Username != "default" {
			authCmd = fmt.Sprintf("AUTH %s %s\r\n", service.Credentials.Username, service.Credentials.Password)
		} else {
			authCmd = fmt.Sprintf("AUTH %s\r\n", service.Credentials.Password)
		}

		if _, err = conn.Write([]byte(authCmd)); err != nil {
			return "N/A"
		}
		response, err := reader.ReadString('\n')
		if err != nil || !strings.HasPrefix(response, "+OK") {
			if strings.Contains(authCmd, " ") {
				authCmd = fmt.Sprintf("AUTH %s\r\n", service.Credentials.Password)
				if _, err = conn.Write([]byte(authCmd)); err == nil {
					response, err = reader.ReadString('\n')
				}
			}
			if err != nil || !strings.HasPrefix(response, "+OK") {
				return "N/A"
			}
		}
		if err := conn.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
			return "N/A"
		}
	}

	if _, err = conn.Write([]byte("INFO server\r\n")); err == nil {
		response, err := reader.ReadString('\n')
		if err == nil && strings.HasPrefix(response, "$") {
			infoData, _ := io.ReadAll(io.LimitReader(reader, 4096))
			infoStr := string(infoData)
			re := regexp.MustCompile(`(redis_version|valkey_version):([0-9]+\.[0-9]+\.[0-9]+)`)
			matches := re.FindStringSubmatch(infoStr)
			if len(matches) >= 3 {
				return matches[2]
			}
		}
	}

	return "N/A"
}

// getHTTPVersion fetches the version from a service's /service endpoint.
func getHTTPVersion(service config.ServiceDefinition) string {
	report, err := GetServiceReport(service)
	if err != nil {
		return "N/A"
	}
	return report.Version.Str
}
