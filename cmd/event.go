package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/ui"
	"github.com/EasterCompany/dex-cli/utils"
)

func handleDefaultEventOutput() error {
	ui.PrintHeader("Event Command Usage")
	ui.PrintInfo("event service           | Show the raw status from the /service endpoint")
	ui.PrintInfo("event log               | Human-readable event logs")
	ui.PrintInfo("                        | -n <count> (default 20)")
	ui.PrintInfo("                        | -t <type>  (e.g., system.test.completed)")
	ui.PrintInfo("event guardian status   | Show current guardian protocol timers")
	ui.PrintInfo("event guardian reset    | Reset guardian protocol timers")
	ui.PrintInfo("event delete <pattern>  | Delete events matching a pattern")
	return nil
}

func handleGuardianStatus() error {
	def, err := config.Resolve("event")
	if err != nil {
		return err
	}

	status, _, err := utils.GetHTTPBody(def.GetHTTP("/agent/status"))
	if err != nil {
		return fmt.Errorf("failed to get guardian status: %w", err)
	}

	ui.PrintCodeBlockFromBytes(status, "guardian-status", "json")
	return nil
}

func handleGuardianReset(args []string) error {
	protocol := "all"
	if len(args) > 0 {
		protocol = args[0]
	}

	def, err := config.Resolve("event")
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s?protocol=%s", def.GetHTTP("/guardian/reset"), protocol)
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to reset guardian: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("guardian reset failed with status: %d", resp.StatusCode)
	}

	ui.PrintSuccess(fmt.Sprintf("Successfully reset %s guardian timer", protocol))
	return nil
}

func handleEventGuardian(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("missing guardian subcommand. Usage: event guardian [status|reset]")
	}

	sub := args[0]
	switch sub {
	case "status":
		return handleGuardianStatus()
	case "reset":
		return handleGuardianReset(args[1:])
	default:
		return fmt.Errorf("unknown guardian subcommand: %s", sub)
	}
}

func handleEventServiceStatus() error {
	def, err := config.Resolve("event")
	if err != nil {
		return err
	}

	status, _, err := utils.GetHTTPBody(def.GetHTTP("/service"))
	if err != nil {
		return fmt.Errorf("failed to get event service status: %w", err)
	}

	ui.PrintCodeBlockFromBytes(status, "event-service", "json")
	return nil
}

func handleEventDelete(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("missing pattern for event delete. Usage: event delete <pattern>")
	}

	if len(args) == 1 && (strings.EqualFold(args[0], "all") || args[0] == "--all") {
		args = []string{"*"}
	}

	eventServicePath, err := config.ExpandPath("~/Dexter/bin/dex-event-service")
	if err != nil {
		return err
	}

	cmdArgs := append([]string{"-delete"}, args...)
	cmd := exec.Command(eventServicePath, cmdArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}

func handleEventLog(args []string) error {
	def, err := config.Resolve("event")
	if err != nil {
		return err
	}

	limit := "20"
	filterType := ""
	for i, arg := range args {
		if arg == "-n" && i+1 < len(args) {
			limit = args[i+1]
		}
		if (arg == "-t" || arg == "--type") && i+1 < len(args) {
			filterType = args[i+1]
		}
	}

	url := fmt.Sprintf("%s?ml=%s&format=json", def.GetHTTP("/events"), limit)
	if filterType != "" {
		url += fmt.Sprintf("&event.type=%s", filterType)
	}

	body, _, err := utils.GetHTTPBody(url)
	if err != nil {
		return fmt.Errorf("failed to get event logs: %w", err)
	}

	var response struct {
		Events []struct {
			Service   string          `json:"service"`
			Timestamp int64           `json:"timestamp"`
			Event     json.RawMessage `json:"event"`
		} `json:"events"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		ui.PrintSubHeader("Last Events (Raw Text)")
		fmt.Println(string(body))
		return nil
	}

	if len(response.Events) == 0 {
		ui.PrintInfo("No events found matching criteria.")
		return nil
	}

	ui.PrintSubHeader(fmt.Sprintf("Last %d Events", len(response.Events)))

	for i := len(response.Events) - 1; i >= 0; i-- {
		e := response.Events[i]
		var data map[string]interface{}
		_ = json.Unmarshal(e.Event, &data)

		t := time.Unix(e.Timestamp, 0).Format("15:04:05")
		eventType, _ := data["type"].(string)

		color := ui.ColorCyan
		if strings.HasPrefix(eventType, "messaging") || strings.Contains(eventType, "message") {
			color = ui.ColorBlue
		} else if strings.HasPrefix(eventType, "system.analysis") || strings.HasPrefix(eventType, "analysis") || strings.HasPrefix(eventType, "engagement") {
			color = ui.ColorPurple
		} else if strings.HasPrefix(eventType, "error") || strings.Contains(eventType, "fail") {
			color = ui.ColorRed
		} else if strings.HasPrefix(eventType, "system.cli") || strings.HasPrefix(eventType, "system.build") || strings.HasPrefix(eventType, "system.test") {
			color = ui.ColorBrightRed
		} else if strings.HasPrefix(eventType, "system.roadmap") || strings.HasPrefix(eventType, "system.process") {
			color = ui.ColorGreen
		}

		summary := eventType
		switch eventType {
		case "messaging.user.sent_message":
			summary = fmt.Sprintf("%s: %s", data["user_name"], data["content"])
		case "messaging.bot.sent_message":
			summary = fmt.Sprintf("Dexter: %s", data["content"])
		case "system.cli.command":
			summary = fmt.Sprintf("CMD: dex %v %v (%v)", data["command"], data["args"], data["status"])
		case "system.cli.status":
			summary = fmt.Sprintf("STATUS: %v", data["message"])
		case "system.test.completed":
			summary = fmt.Sprintf("TESTS: %v (%v)", data["service_name"], data["duration"])
		case "system.roadmap.created":
			summary = fmt.Sprintf("ROADMAP+: %v", data["content"])
		case "system.roadmap.updated":
			summary = fmt.Sprintf("ROADMAP~: %v -> %v", data["id"], data["state"])
		case "system.process.registered":
			summary = fmt.Sprintf("PROC+: %v (%v)", data["id"], data["state"])
		case "system.process.unregistered":
			summary = fmt.Sprintf("PROC-: %v", data["id"])
		case "log_entry":
			summary = fmt.Sprintf("[%v] %v", data["level"], data["message"])
		}

		fmt.Printf("%s %s%-15s%s | %s%s%s\n",
			ui.ColorDarkGray+t+ui.ColorReset,
			ui.ColorDarkGray, e.Service, ui.ColorReset,
			color, summary, ui.ColorReset)
	}

	return nil
}

func Event(args []string) error {
	if len(args) == 0 {
		return handleDefaultEventOutput()
	}

	subcommand := args[0]
	switch subcommand {
	case "service":
		return handleEventServiceStatus()
	case "guardian":
		return handleEventGuardian(args[1:])
	case "delete":
		return handleEventDelete(args[1:])
	case "log":
		return handleEventLog(args[1:])
	default:
		return fmt.Errorf("unknown event subcommand: %s", subcommand)
	}
}
