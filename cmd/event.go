package cmd

import (
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
	ui.PrintInfo("event service          | Show the raw status from the /service endpoint")
	ui.PrintInfo("event log              | Show the last 10 human-readable event logs")
	ui.PrintInfo("event analyst status   | Show current analyst tier timers")
	ui.PrintInfo("event analyst reset    | Reset analyst strategist timer")
	ui.PrintInfo("event delete <pattern> | Delete events matching a pattern (e.g., 'event delete all')")
	return nil
}

func handleAnalystStatus() error {
	def, err := config.Resolve("event")
	if err != nil {
		return err
	}

	status, _, err := utils.GetHTTPBody(def.GetHTTP("/analyst/status"))
	if err != nil {
		return fmt.Errorf("failed to get analyst status: %w", err)
	}

	ui.PrintCodeBlockFromBytes(status, "analyst-status", "json")
	return nil
}

func handleAnalystReset(args []string) error {
	tier := "strategist"
	if len(args) > 0 {
		tier = args[0]
	}

	def, err := config.Resolve("event")
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s?tier=%s", def.GetHTTP("/analyst/reset"), tier)
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to reset analyst: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("analyst reset failed with status: %d", resp.StatusCode)
	}

	ui.PrintSuccess(fmt.Sprintf("Successfully reset %s analyst timer", tier))
	return nil
}

func handleEventAnalyst(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("missing analyst subcommand. Usage: event analyst [status|reset]")
	}

	sub := args[0]
	switch sub {
	case "status":
		return handleAnalystStatus()
	case "reset":
		return handleAnalystReset(args[1:])
	default:
		return fmt.Errorf("unknown analyst subcommand: %s", sub)
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

	// Handle "all" keyword to safely delete everything without shell expansion issues with "*"
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
	cmd.Stdin = os.Stdin // Connect stdin for interactive confirmation

	return cmd.Run()
}

func handleEventLog() error {
	def, err := config.Resolve("event")
	if err != nil {
		return err
	}

	logs, _, err := utils.GetHTTPBody(def.GetHTTP("/events?ml=10&format=text"))
	if err != nil {
		return fmt.Errorf("failed to get event logs: %w", err)
	}

	if strings.TrimSpace(string(logs)) == "" {
		ui.PrintInfo("No events found.")
		return nil
	}

	ui.PrintSubHeader("Last 10 Events")
	fmt.Println(string(logs))
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
	case "analyst":
		return handleEventAnalyst(args[1:])
	case "delete":
		return handleEventDelete(args[1:])
	case "log":
		return handleEventLog()
	default:
		return fmt.Errorf("unknown event subcommand: %s", subcommand)
	}
}
