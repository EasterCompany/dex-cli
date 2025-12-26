package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/ui"
	"github.com/EasterCompany/dex-cli/utils"
)

func Guardian(args []string) error {
	ui.PrintHeader("Guardian Analyst")
	ui.PrintInfo("Initializing Tier 1 Analysis...")

	def, err := config.Resolve("event")
	if err != nil {
		return fmt.Errorf("failed to resolve event service: %w", err)
	}

	// Helper to report status to the dashboard
	reportStatus := func(state string, activeTier string) {
		client := &http.Client{Timeout: 2 * time.Second}

		// 1. Report Active Process
		processPayload := map[string]string{
			"id":    "system-analyst",
			"state": state,
		}
		pBody, _ := json.Marshal(processPayload)
		req, _ := http.NewRequest("POST", def.GetHTTP("/processes"), bytes.NewBuffer(pBody))
		req.Header.Set("Content-Type", "application/json")
		_, _ = client.Do(req)

		// 2. Report Active Analyst Tier
		tierPayload := map[string]string{
			"active_tier": activeTier,
		}
		tBody, _ := json.Marshal(tierPayload)
		tReq, _ := http.NewRequest("PATCH", def.GetHTTP("/analyst/status"), bytes.NewBuffer(tBody))
		tReq.Header.Set("Content-Type", "application/json")
		_, _ = client.Do(tReq)
	}

	// Helper to clear status from the dashboard
	clearStatus := func() {
		client := &http.Client{Timeout: 2 * time.Second}

		// 1. Remove Process
		req, _ := http.NewRequest("DELETE", def.GetHTTP("/processes/system-analyst"), nil)
		_, _ = client.Do(req)

		// 2. Clear Active Tier
		tierPayload := map[string]string{
			"active_tier": "",
		}
		tBody, _ := json.Marshal(tierPayload)
		tReq, _ := http.NewRequest("PATCH", def.GetHTTP("/analyst/status"), bytes.NewBuffer(tBody))
		tReq.Header.Set("Content-Type", "application/json")
		_, _ = client.Do(tReq)
	}

	// Report starting
	reportStatus("Tier 1: Guardian Analysis", "guardian")
	defer clearStatus()

	// 1. Fetch System Context
	ui.PrintRunningStatus("Fetching system context...")

	// Fetch System Monitor
	monitorData, _, err := utils.GetHTTPBody(def.GetHTTP("/system_monitor"))
	if err != nil {
		ui.PrintWarning(fmt.Sprintf("Failed to fetch system monitor: %v", err))
		monitorData = []byte("{}")
	}

	// Fetch Recent Events
	eventsData, _, err := utils.GetHTTPBody(def.GetHTTP("/events?ml=20&format=json"))
	if err != nil {
		ui.PrintWarning(fmt.Sprintf("Failed to fetch events: %v", err))
		eventsData = []byte("[]")
	}

	// Fetch Recent Logs
	logsData, _, err := utils.GetHTTPBody(def.GetHTTP("/logs?ml=50"))
	if err != nil {
		ui.PrintWarning(fmt.Sprintf("Failed to fetch logs: %v", err))
		logsData = []byte("[]")
	}

	// 2. Construct Prompt
	prompt := `### SYSTEM_ROLE: TECHNICAL_AUDIT_BOT
### TASK: Audit system status and logs.
### FORMAT: PURE MARKDOWN ONLY. NO PROSE.

### FEW-SHOT EXAMPLES:

INPUT: (Offline services, error logs)
OUTPUT:
# Service Failure Alert
**Type**: alert
**Priority**: high
**Category**: system
**Affected**: dex-tts-service
**Related IDs**: none

## Summary
The TTS service is offline due to CUDA memory exhaustion.

## Content
Logs show "RuntimeError: CUDA out of memory". Process 12345.
---


INPUT: (All OK)
OUTPUT: No significant insights found.

### END EXAMPLES.

### CURRENT DATA TO AUDIT:
`
	prompt += fmt.Sprintf("\nSYSTEM MONITOR:\n%s\n", string(monitorData))
	prompt += fmt.Sprintf("\nRECENT EVENTS:\n%s\n", string(eventsData))
	prompt += fmt.Sprintf("\nRECENT LOGS:\n%s\n", string(logsData))

	// 3. Run Chat
	ui.PrintRunningStatus("Running Guardian Analysis...")
	fmt.Println() // Spacing

	messages := []utils.Message{
		{Role: "user", Content: prompt},
	}

	fullResponse := ""
	err = utils.ChatStream("dex-analyst-guardian", messages, func(chunk string) {
		fmt.Print(chunk)
		fullResponse += chunk
	})
	fmt.Println() // Newline after stream
	if err != nil {
		return fmt.Errorf("analysis failed: %w", err)
	}

	// 4. Reset Timer
	ui.PrintRunningStatus("Resetting Guardian timer...")
	url := def.GetHTTP("/analyst/reset?tier=guardian")
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create reset request: %w", err)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to reset analyst: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		ui.PrintWarning(fmt.Sprintf("Analyst reset returned status: %d", resp.StatusCode))
	} else {
		ui.PrintSuccess("Guardian timer reset.")
	}

	// Helper to send a CLI event about this run
	utils.SendEvent("system.analysis.audit", map[string]interface{}{
		"tier":       "guardian",
		"model":      "dex-analyst-guardian",
		"raw_output": fullResponse,
		"raw_input":  "CLI Manual Trigger",
	})

	return nil
}
