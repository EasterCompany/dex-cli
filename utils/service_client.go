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
	body, statusCode, err := GetHTTPBody(service.GetHTTP("/service"))
	if err != nil {
		return "N/A", fmt.Errorf("failed to connect to %s: %w", service.ShortName, err)
	}

	status := strings.TrimSpace(string(body))

	if statusCode != http.StatusOK {
		return "N/A", fmt.Errorf("%s returned status: %d - %s", service.ShortName, statusCode, status)
	}

	return status, nil
}

// GetHTTPBody fetches the raw body from an HTTP endpoint.
func GetHTTPBody(url string) ([]byte, int, error) {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}

	return body, resp.StatusCode, nil
}
