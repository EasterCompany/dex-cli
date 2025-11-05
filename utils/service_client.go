package utils

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/EasterCompany/dex-cli/config"
)

// GetHTTPVersion fetches the version from a service's HTTP endpoint.
func GetHTTPVersion(service config.ServiceDefinition) (string, error) {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(service.GetHTTP("/service"))
	if err != nil {
		return "N/A", fmt.Errorf("failed to connect to %s: %w", service.ShortName, err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "N/A", fmt.Errorf("failed to read response from %s: %w", service.ShortName, err)
	}

	status := strings.TrimSpace(string(body))

	if resp.StatusCode != http.StatusOK {
		return "N/A", fmt.Errorf("%s returned status: %s - %s", service.ShortName, resp.Status, status)
	}

	return status, nil
}
