package utils

import (
	"bytes"
	"encoding/json"
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

// GetHTTPServiceReport retrieves the full JSON service report from /service endpoint
func GetHTTPServiceReport(service config.ServiceDefinition) (string, error) {
	url := service.GetHTTP("/service")
	body, statusCode, err := GetHTTPBody(url)
	if err != nil {
		return "", fmt.Errorf("failed to connect to %s: %w", service.ShortName, err)
	}

	if statusCode != http.StatusOK {
		return "", fmt.Errorf("%s returned status: %d", service.ShortName, statusCode)
	}

	return string(body), nil
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

// SendEvent sends an event to the event service and waits for completion.
// In a CLI tool, this must be synchronous to ensure events are sent before the process exits.
func SendEvent(eventType string, eventData map[string]interface{}) {
	defer func() {
		if r := recover(); r != nil {
			// Prevent crash on panic
			fmt.Printf("Recovered from panic in SendEvent: %v\n", r)
		}
	}()

	// Get Event Service URL
	var url string
	def := config.GetServiceDefinition("dex-event-service")
	if def.ID == "" {
		url = "http://localhost:8100/events" // Corrected default port
	} else {
		url = def.GetHTTP("/events")
	}

	// Basic event structure
	eventData["type"] = eventType

	// Ensure timestamp exists
	if _, ok := eventData["timestamp"]; !ok {
		eventData["timestamp"] = time.Now().Format(time.RFC3339)
	}

	requestBody := map[string]interface{}{
		"service": "dex-cli",
		"event":   eventData,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return
	}

	client := &http.Client{
		Timeout: 1 * time.Second, // Reduced timeout for CLI responsiveness
	}

	resp, err := client.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return
	}
	defer func() { _ = resp.Body.Close() }()
}
