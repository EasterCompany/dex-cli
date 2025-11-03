package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// formatUptime formats a duration into a human-readable string
func formatUptime(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if hours > 0 {
		return fmt.Sprintf("%dh%dm", hours, minutes)
	} else if minutes > 0 {
		return fmt.Sprintf("%dm%ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}

// fetchLatestVersion fetches the latest version tag from easter.company
func fetchLatestVersion() (string, error) {
	resp, err := http.Get("https://easter.company/tags/dex-cli.json")
	if err != nil {
		return "", fmt.Errorf("failed to fetch latest version: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	var tags struct {
		Latest string `json:"latest"`
	}
	if err := json.Unmarshal(body, &tags); err != nil {
		return "", fmt.Errorf("failed to parse JSON: %w", err)
	}

	return tags.Latest, nil
}
