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
	// Append ?format=version to get the raw version string
	url := service.GetHTTP("/service") + "?format=version"
	body, statusCode, err := GetHTTPBody(url)
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

// FetchURL fetches the raw body from a URL with a custom timeout.
func FetchURL(url string, timeout time.Duration) (string, error) {
	client := &http.Client{
		Timeout: timeout,
	}

	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}
