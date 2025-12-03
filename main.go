package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

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
	// Set up signal handler to ensure clean exit with blank line
	setupSignalHandler()

	cmd.RunningVersion = version
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
		runCommand(func() error { return cmd.Service(command) })

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

	case "test":
		runCommand(func() error { return cmd.Test(os.Args[2:]) })

	case "python":
		runCommand(func() error { return utils.Python(os.Args[2:]) })

	case "bun":
		runCommand(func() error { return cmd.Bun(os.Args[2:]) })

	case "bunx":
		runCommand(func() error { return cmd.Bunx(os.Args[2:]) })

	case "ollama":
		runCommand(func() error { return cmd.Ollama(os.Args[2:]) })

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
	fmt.Println()
	if err := commandFunc(); err != nil {
		ui.PrintError(fmt.Sprintf("Error: %v", err))
		fmt.Println() // Add padding at the end
		os.Exit(1)
	}
	fmt.Println()
}

func setupSignalHandler() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		<-sigChan
		// Clear any progress indicators and print final blank line
		ui.ClearLine()
		fmt.Println()
		os.Exit(130) // Standard exit code for SIGINT (128 + 2)
	}()
}

func printUsage() {
	ui.PrintHeader("DEX")
	ui.PrintSection("A cli program for interfacing with local and/or remote dexter services as a user and/or developer.")
	ui.PrintSubHeader("Local/User System Commands")
	if config.IsCommandAvailable("system") {
		ui.PrintInfo("system     | Show system info and manage packages")
	}
	if config.IsCommandAvailable("config") {
		ui.PrintInfo("config     | <service> [field...] Show service config or a specific field")
		ui.PrintInfo("           | reset                Reset service-map.json to default configuration")
	}
	if config.IsCommandAvailable("cache") {
		ui.PrintInfo("cache      | [clear|list] Manage the local cache")
	}
	if config.IsCommandAvailable("logs") {
		ui.PrintInfo("logs       | <service> [-f] View service logs")
	}
	ui.PrintSubHeader("Developer Lifecycle Commands")
	if config.IsCommandAvailable("test") {
		ui.PrintInfo("test       | Test services")
	}
	if config.IsCommandAvailable("build") {
		ui.PrintInfo("build      | Build and install cli and services")
	}
	if config.IsCommandAvailable("update") {
		ui.PrintInfo("update     | Update cli and services")
	}
	ui.PrintSubHeader("Global Service Management Commands")
	if config.IsCommandAvailable("status") {
		ui.PrintInfo("status     | Checks the status of cli and services")
	}
	if config.IsCommandAvailable("add") {
		ui.PrintInfo("add        | Add (install) a service")
	}
	if config.IsCommandAvailable("remove") {
		ui.PrintInfo("remove     | Remove (uninstall) a service")
	}
	if config.IsCommandAvailable("start") {
		ui.PrintInfo("start      | Start all manageable services")
	}
	if config.IsCommandAvailable("stop") {
		ui.PrintInfo("stop       | Stop all manageable services")
	}
	if config.IsCommandAvailable("restart") {
		ui.PrintInfo("restart    | Restart all manageable services")
	}
	ui.PrintSubHeader("Proxy Commands")
	if config.IsCommandAvailable("python") {
		ui.PrintInfo("python     | Access the python virtual environment")
	}
	if config.IsCommandAvailable("bun") {
		ui.PrintInfo("bun        | Access the system bun executable")
	}
	if config.IsCommandAvailable("bunx") {
		ui.PrintInfo("bunx       | Access the system bunx executable")
	}
	if config.IsCommandAvailable("ollama") {
		ui.PrintInfo("ollama     | Access the system ollama executable")
	}
	if config.IsCommandAvailable("whisper") {
		ui.PrintInfo("whisper    | Speech-to-text transcription with Whisper")
	}
	ui.PrintSubHeader("Service Commands")
	if config.IsCommandAvailable("event") {
		ui.PrintInfo("event      | Interact with the event service for this instance")
	}
	if config.IsCommandAvailable("model") {
		ui.PrintInfo("model      | Interact with the model service for this instance")
	}
	if config.IsCommandAvailable("discord") {
		ui.PrintInfo("discord    | Interact with the discord service for this instance")
	}
	ui.PrintSubHeader("CLI Basic commands")
	ui.PrintInfo("version    Show version information")
	ui.PrintInfo("help       Show this help message")
	fmt.Println()
}
