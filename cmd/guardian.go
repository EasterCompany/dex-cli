package cmd

import (
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

	// 1. Fetch System Context

	ui.PrintRunningStatus("Fetching system context...")

	def, err := config.Resolve("event")

	if err != nil {

		return fmt.Errorf("failed to resolve event service: %w", err)

	}

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

	// 5. Check if we need to emit a notification event based on the output?
	// The real service does this. The CLI just displays it.
	// But the user said: "until the return format is given or the attention ends."
	// The streaming handles "attention ends".

	// If the response contains a report, we *could* post it back to the event service?
	// But for now, just local display.

	// Helper to send a CLI event about this run
	utils.SendEvent("system.analysis.audit", map[string]interface{}{
		"tier":       "guardian",
		"model":      "dex-analyst-guardian",
		"raw_output": fullResponse,
		"raw_input":  "CLI Manual Trigger",
	})

	return nil
}
