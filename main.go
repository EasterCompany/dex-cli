package main

import (
	"fmt"
	"os"
	"time"

	"github.com/EasterCompany/dex-cli/cmd"
	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/ui"
	"github.com/EasterCompany/dex-cli/utils"
)

var (
	version   string
	branch    string
	commit    string
	buildDate string
	buildYear string
	buildHash string
)

func main() {
	cmd.RunningVersion = version

	// Globally filter out --no-event flag
	newArgs := []string{}
	for _, arg := range os.Args {
		if arg == "--no-event" || arg == "--json" {
			utils.SuppressEvents = true
		}

		if arg != "--no-event" {
			newArgs = append(newArgs, arg)
		}
	}
	os.Args = newArgs

	if len(os.Args) > 1 && os.Args[1] != "version" {
		command := os.Args[1]
		isVerboseCommand := command == "build" || command == "update" || command == "test"
		if err := utils.EnsurePythonVenv(!isVerboseCommand); err != nil {
			fmt.Println() // Add padding at the start
			ui.PrintError(fmt.Sprintf("Error ensuring Python environment: %v", err))
			fmt.Println() // Add padding at the end
			os.Exit(1)
		}
	}

	if err := config.EnsureDirectoryStructure(); err != nil {
		fmt.Println() // Add padding at the start
		ui.PrintError(fmt.Sprintf("Error ensuring directory structure: %v", err))
		fmt.Println() // Add padding at the end
		os.Exit(1)
	}

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	if !config.IsCommandAvailable(command) {
		fmt.Println() // Add padding at the start
		ui.PrintError(fmt.Sprintf("Command '%s' is not available", command))
		ui.PrintInfo("This command requires certain conditions to be met.")
		printUsage()
		os.Exit(1)
	}

	switch command {
	case "update":
		runCommand(func() error { return cmd.Update(os.Args[2:]) })

	case "verify":
		runCommand(func() error { return cmd.Verify() })

	case "system":
		runCommand(func() error { return cmd.System(os.Args[2:]) })

	case "version", "-v", "--version":
		jsonOutput := false
		for _, arg := range os.Args {
			if arg == "--json" {
				jsonOutput = true
				break
			}
		}
		fmt.Println() // Add padding at the start
		cmd.Version(jsonOutput, version, branch, commit, buildDate, buildYear, buildHash)
		fmt.Println() // Add padding at the end

	case "build":
		runCommand(func() error { return cmd.Build(os.Args[2:]) })

	case "start", "stop", "restart":
		runCommand(func() error { return cmd.Service(command, os.Args[2:]) })

	case "status":
		service := "all"
		if len(os.Args) > 2 {
			service = os.Args[2]
		}
		runCommand(func() error { return cmd.Status(service) })

	case "logs":
		follow := false
		args := os.Args[2:]
		for i, arg := range args {
			if arg == "-f" {
				follow = true
				args = append(args[:i], args[i+1:]...)
				break
			}
		}
		runCommand(func() error { return cmd.Logs(args, follow) })

	case "fmt":

		runCommand(func() error { return cmd.Fmt(os.Args[2:]) })

	case "lint":

		runCommand(func() error { return cmd.Lint(os.Args[2:]) })

	case "ollama":

		runCommand(func() error { return cmd.Ollama(os.Args[2:]) })

	case "test":
		runCommand(func() error { return cmd.Test(os.Args[2:]) })

	case "python":
		runCommand(func() error { return utils.Python(os.Args[2:]) })

	case "bun":
		runCommand(func() error { return cmd.Bun(os.Args[2:]) })

	case "bunx":
		runCommand(func() error { return cmd.Bunx(os.Args[2:]) })

	case "add":
		runCommand(func() error { return cmd.Add(os.Args[2:]) })

	case "remove":
		runCommand(func() error { return cmd.Remove(os.Args[2:]) })

	case "cache":
		runCommand(func() error { return cmd.Cache(os.Args[2:]) })

	case "event":
		runCommand(func() error { return cmd.Event(os.Args[2:]) })

	case "discord":
		runCommand(func() error { return cmd.Discord(os.Args[2:]) })

	case "config":
		// config command can have subcommands like 'reset'
		runCommand(func() error { return cmd.Config(os.Args[2:]) })

	case "whisper":
		runCommand(func() error { return cmd.Whisper(os.Args[2:]) })

	case "agent":
		runCommand(func() error { return cmd.Agent(os.Args[2:]) })

	case "courier":
		runCommand(func() error { return cmd.Courier(os.Args[2:]) })

	case "serve": // New serve command
		runCommand(func() error { return cmd.Serve(os.Args[2:], version, branch, commit, buildDate) })

	case "help", "-h", "--help":
		printUsage()

	default:
		fmt.Println()
		ui.PrintError(fmt.Sprintf("Unknown command: %s", command))
		fmt.Println()
		os.Exit(1)
	}
}

func runCommand(commandFunc func() error) {
	command := "unknown"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}
	args := ""
	if len(os.Args) > 2 {
		args = fmt.Sprintf("%v", os.Args[2:])
	}

	// Emit Command Started Event
	utils.SendEvent("system.cli.command", map[string]interface{}{
		"command": command,
		"args":    args,
		"status":  "started",
	})

	// Start capturing UI output
	ui.StartCapturing()

	// Special handling for build/update commands to set Discord status
	if command == "build" || command == "update" {
		utils.SendEvent("system.cli.status", map[string]interface{}{
			"status":  "dnd",
			"message": fmt.Sprintf("%s in progress...", command),
		})
	}

	fmt.Println()
	start := time.Now()
	err := commandFunc()
	duration := time.Since(start).String()
	ui.StopCapturing()
	capturedOutput := ui.GetCapturedOutput()

	if err != nil {
		// Emit Command Error Event (unless it's build which handles its own)
		if command != "build" {
			utils.SendEvent("system.cli.command", map[string]interface{}{
				"command":   command,
				"args":      args,
				"status":    "error",
				"output":    capturedOutput + "\nError: " + err.Error(),
				"duration":  duration,
				"exit_code": 1,
			})
		}

		// Reset status if it was update
		if command == "update" {
			utils.SendEvent("system.cli.status", map[string]interface{}{
				"status":  "online",
				"message": "Operation failed",
			})
		}

		ui.PrintError(fmt.Sprintf("Error: %v", err))
		fmt.Println() // Add padding at the end
		os.Exit(1)
	}

	// Emit Command Success Event (unless it's build)
	if command != "build" {
		utils.SendEvent("system.cli.command", map[string]interface{}{
			"command":   command,
			"args":      args,
			"status":    "success",
			"output":    capturedOutput,
			"duration":  duration,
			"exit_code": 0,
		})
	}

	// Reset status on success
	if command == "update" {
		utils.SendEvent("system.cli.status", map[string]interface{}{
			"status":  "online",
			"message": "Operation complete",
		})
	}

	fmt.Println()
}

func printUsage() {
	ui.PrintHeader("DEX - The Digital Executive Interface")
	ui.PrintSection("A unified CLI for managing the Easter Company AI ecosystem.")

	ui.PrintSubHeader("CORE LIFECYCLE")
	ui.PrintKeyValBlock("build", []ui.KeyVal{
		{Key: "Usage", Value: "dex build [major|minor|patch] [-f|--force]"},
		{Key: "Desc", Value: "Build and install services from local source."},
		{Key: "Args", Value: "Increment version: 'patch' (default), 'minor', or 'major'."},
		{Key: "Flags", Value: "--force: Rebuild all services even without changes."},
	})
	ui.PrintKeyValBlock("update", []ui.KeyVal{
		{Key: "Usage", Value: "dex update"},
		{Key: "Desc", Value: "Update CLI and services. In DEV mode: Rebuilds from source. In USER mode: Downloads binaries."},
	})
	ui.PrintKeyValBlock("test", []ui.KeyVal{
		{Key: "Usage", Value: "dex test [service] [--models]"},
		{Key: "Desc", Value: "Run format, lint, and unit tests."},
		{Key: "Args", Value: "[service]: specific service to test. Defaults to all."},
		{Key: "Flags", Value: "--models: Include comprehensive model tests (slow)."},
	})

	ui.PrintSubHeader("SERVICE MANAGEMENT")
	ui.PrintKeyValBlock("start/stop/restart", []ui.KeyVal{
		{Key: "Usage", Value: "dex [start|stop|restart] <service|all>"},
		{Key: "Desc", Value: "Manage background systemd services."},
	})
	ui.PrintKeyValBlock("status", []ui.KeyVal{
		{Key: "Usage", Value: "dex status [service|all]"},
		{Key: "Desc", Value: "Check connectivity and health of services."},
	})
	ui.PrintKeyValBlock("logs", []ui.KeyVal{
		{Key: "Usage", Value: "dex logs <service> [-f]"},
		{Key: "Desc", Value: "View service logs."},
		{Key: "Flags", Value: "-f: Follow log output in real-time."},
	})

	ui.PrintSubHeader("SYSTEM & CONFIGURATION")
	ui.PrintKeyValBlock("system", []ui.KeyVal{
		{Key: "Usage", Value: "dex system [command]"},
		{Key: "Commands", Value: "info (default): Show hardware/software specs."},
		{Key: "", Value: "scan: Re-scan hardware and update config."},
		{Key: "", Value: "validate: Check for missing required packages."},
		{Key: "", Value: "install [pkg]: Install missing system package(s)."},
		{Key: "", Value: "upgrade [pkg]: Upgrade installed system package(s)."},
	})
	ui.PrintKeyValBlock("config", []ui.KeyVal{
		{Key: "Usage", Value: "dex config <service> [field] | reset"},
		{Key: "Desc", Value: "View or manage service configuration (service-map.json)."},
		{Key: "Examples", Value: "'dex config event http_port' or 'dex config reset'."},
	})
	ui.PrintKeyValBlock("cache", []ui.KeyVal{
		{Key: "Usage", Value: "dex cache [clear|list]"},
		{Key: "Desc", Value: "Manage the local CLI cache (GitHub access, etc)."},
	})

	ui.PrintSubHeader("INTELLIGENCE & ANALYSIS")
	ui.PrintKeyValBlock("agent", []ui.KeyVal{
		{Key: "Usage", Value: "dex agent <name> [run|reset] [-f|--force]"},
		{Key: "Desc", Value: "Manage Dexter Agents (guardian, analyzer)."},
		{Key: "Flags", Value: "--force: Bypass checks (e.g., idle/cooldown for guardian)."},
	})
	ui.PrintKeyValBlock("courier", []ui.KeyVal{
		{Key: "Usage", Value: "dex courier [run]"},
		{Key: "Cmd", Value: "courier"},
		{Key: "Desc", Value: "Run the Courier Protocol to execute active research tasks."},
	})
	ui.PrintKeyValBlock("event", []ui.KeyVal{
		{Key: "Usage", Value: "dex event [subcommand]"},
		{Key: "Desc", Value: "Interact with the Event Service."},
		{Key: "Subcommands", Value: "log [-n count] [-t type]: View raw event log."},
		{Key: "", Value: "service: Show raw service status JSON."},
		{Key: "", Value: "guardian status: Show guardian timers."},
		{Key: "", Value: "guardian reset: Reset guardian timers."},
		{Key: "", Value: "delete <pattern>: Delete events matching pattern."},
	})
	ui.PrintKeyValBlock("discord", []ui.KeyVal{
		{Key: "Usage", Value: "dex discord [subcommand]"},
		{Key: "Desc", Value: "Interact with the Discord Service."},
		{Key: "Subcommands", Value: "contacts: List members and levels."},
		{Key: "", Value: "channels: List guild channel structure."},
		{Key: "", Value: "quiet [on|off]: Toggle quiet mode."},
		{Key: "", Value: "service: Show raw service status JSON."},
	})

	ui.PrintSubHeader("TOOLS & UTILITIES")
	ui.PrintKeyValBlock("ollama", []ui.KeyVal{
		{Key: "Usage", Value: "dex ollama [pull|list|rm]"},
		{Key: "Desc", Value: "Manage local LLM models."},
	})
	ui.PrintKeyValBlock("fmt", []ui.KeyVal{
		{Key: "Usage", Value: "dex fmt"},
		{Key: "Desc", Value: "Format all source code (Go, JS, HTML, CSS, etc.)."},
	})
	ui.PrintKeyValBlock("lint", []ui.KeyVal{
		{Key: "Usage", Value: "dex lint"},
		{Key: "Desc", Value: "Run code quality checks (ESLint, Stylelint, HTMLHint, Go)."},
	})
	ui.PrintKeyValBlock("whisper", []ui.KeyVal{
		{Key: "Usage", Value: "dex whisper [file]"},
		{Key: "Desc", Value: "Transcribe audio file using local Whisper model."},
	})
	ui.PrintKeyValBlock("serve", []ui.KeyVal{
		{Key: "Usage", Value: "dex serve -d <dir> -p <port>"},
		{Key: "Desc", Value: "Serve static files from a directory."},
	})
	ui.PrintKeyValBlock("python/bun", []ui.KeyVal{
		{Key: "Usage", Value: "dex [python|bun|bunx] [args...]"},
		{Key: "Desc", Value: "Run command within the project's environment."},
	})

	ui.PrintSubHeader("PACKAGE MANAGEMENT")
	ui.PrintKeyValBlock("add/remove", []ui.KeyVal{
		{Key: "Usage", Value: "dex [add|remove] <service>"},
		{Key: "Desc", Value: "Install or uninstall a service from the ecosystem."},
	})

	ui.PrintSubHeader("GLOBAL FLAGS")
	ui.PrintInfo("--no-event  | Suppress event emission for the command.")
	ui.PrintInfo("--json      | Output result as JSON where supported.")
	fmt.Println()
}
