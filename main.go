package main

import (
	"fmt"
	"os"

	"github.com/EasterCompany/dex-cli/cmd"
	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/ui"
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
	// Set the running version string from build-time variables.
	cmd.RunningVersion = cmd.FormatVersion(version, branch, commit, buildDate, buildHash)

	if err := cmd.EnsurePythonVenv(version); err != nil {
		ui.PrintError(fmt.Sprintf("Error ensuring Python environment: %v", err))
		os.Exit(1)
	}

	if err := config.EnsureDirectoryStructure(); err != nil {
		ui.PrintError(fmt.Sprintf("Error ensuring directory structure: %v", err))
		os.Exit(1)
	}

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	if !config.IsCommandAvailable(command) {
		ui.PrintError(fmt.Sprintf("Command '%s' is not available", command))
		ui.PrintInfo("This command requires certain conditions to be met.")
		printUsage()
		os.Exit(1)
	}

	switch command {
	case "update":
		runCommand(func() error { return cmd.Update(os.Args[2:], buildYear) })

	case "system":
		runCommand(func() error { return cmd.System(os.Args[2:]) })

	case "version", "-v", "--version":
		fmt.Println() // Add padding at the start
		cmd.Version(version, branch, commit, buildDate, buildYear, buildHash)
		fmt.Println() // Add padding at the end

	case "build":
		runCommand(func() error { return cmd.Build(os.Args[2:]) })

	case "start", "stop", "restart":
		if len(os.Args) < 3 {
			ui.PrintError(fmt.Sprintf("Error: service name required for '%s' command", command))
			os.Exit(1)
		}
		service := os.Args[2]
		runCommand(func() error { return cmd.Service(command, service) })

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
		runCommand(func() error { return cmd.Python(os.Args[2:]) })

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

	case "config":
		runCommand(func() error { return cmd.Config(os.Args[2:]) })

	case "help", "-h", "--help":
		printUsage()

	default:
		ui.PrintError(fmt.Sprintf("Unknown command: %s", command))
		printUsage()
		os.Exit(1)
	}
}

func runCommand(commandFunc func() error) {
	fmt.Println()
	if err := commandFunc(); err != nil {
		ui.PrintError(fmt.Sprintf("Error: %v", err))
		os.Exit(1)
	}
	fmt.Println()
}

func printUsage() {
	ui.PrintInfo("dex <command> [options]")

	if config.IsCommandAvailable("update") {
		ui.PrintInfo("update     Update dex-cli and all services")
	}
	if config.IsCommandAvailable("build") {
		ui.PrintInfo("build      Build and install all services from local source")
	}
	if config.IsCommandAvailable("status") {
		ui.PrintInfo("status     [service] Check the health of one or all services")
	}
	if config.IsCommandAvailable("start") {
		ui.PrintInfo("start      <service> Start a service")
	}
	if config.IsCommandAvailable("stop") {
		ui.PrintInfo("stop       <service> Stop a service")
	}
	if config.IsCommandAvailable("restart") {
		ui.PrintInfo("restart    <service> Restart a service")
	}
	if config.IsCommandAvailable("logs") {
		ui.PrintInfo("logs       <service> [-f] View service logs")
	}
	if config.IsCommandAvailable("test") {
		ui.PrintInfo("test       Run all tests")
	}
	if config.IsCommandAvailable("system") {
		ui.PrintInfo("system     Show system info and manage packages")
	}
	if config.IsCommandAvailable("python") {
		ui.PrintInfo("python     [<subcommand>] [args...] Python virtual environment")
	}
	if config.IsCommandAvailable("bun") {
		ui.PrintInfo("bun        [args...] System's bun executable")
	}
	if config.IsCommandAvailable("bunx") {
		ui.PrintInfo("bunx       [args...] System's bunx executable")
	}
	if config.IsCommandAvailable("add") {
		ui.PrintInfo("add        Add (clone, build, install) a new service")
	}
	if config.IsCommandAvailable("remove") {
		ui.PrintInfo("remove     Uninstall and delete a service")
	}
	if config.IsCommandAvailable("cache") {
		ui.PrintInfo("cache      [clear|list] Manage the local cache")
	}
	if config.IsCommandAvailable("event") {
		ui.PrintInfo("event      Interact with the event service")
	}
	if config.IsCommandAvailable("config") {
		ui.PrintInfo("config     <service> [field...] Show service config or a specific field")
	}
	ui.PrintInfo("version    Show version information")
	ui.PrintInfo("help       Show this help message")
	ui.PrintInfo("Dexter root:        ~/Dexter")
	ui.PrintInfo("EasterCompany root: ~/EasterCompany")
	fmt.Println()
	fmt.Println("™ © 2024 The Easter Company. All rights reserved.")
}
