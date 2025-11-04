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
	cmd := exec.Command("dex", "version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "N/A"
	}
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

// getHTTPVersion fetches the version from a service's /version endpoint.
func getHTTPVersion(service config.ServiceDefinition) string {
	versionURL := service.GetHTTP("/version")
	client := http.Client{
		Timeout: 2 * time.Second,
	}
	resp, err := client.Get(versionURL)
	if err != nil {
		return "N/A"
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "N/A"
	}

	var versionResp map[string]string
	if err := json.Unmarshal(body, &versionResp); err != nil {
		return "N/A"
	}

	parsedVersion, err := git.Parse(versionResp["version"])
	if err != nil {
		return "N/A"
	}

	return parsedVersion.String()
}
