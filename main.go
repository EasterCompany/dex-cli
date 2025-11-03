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
)

func main() {
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

	// Check if command is available based on requirements
	if !config.IsCommandAvailable(command) {
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
		cmd.Version(version, branch, commit, buildDate, buildYear)

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

	case "pull":
		runCommand(func() error { return cmd.Pull(os.Args[2:]) })

	case "help", "-h", "--help":
		printUsage()

	default:
		ui.PrintError(fmt.Sprintf("Unknown command: %s", command))
		printUsage()
		os.Exit(1)
	}
}

// runCommand is a universal wrapper for all commands.
// It executes the command function and prints a blank line at the end.
func runCommand(commandFunc func() error) {
	if err := commandFunc(); err != nil {
		ui.PrintError(fmt.Sprintf("Error: %v", err))
		os.Exit(1)
	}
	fmt.Println()
}

func printUsage() {
	ui.PrintInfo("dex <command> [options]")

	if config.IsCommandAvailable("update") {
		ui.PrintInfo("update     Update dex-cli to latest version")
	}
	if config.IsCommandAvailable("build") {
		ui.PrintInfo("build      <service|all> Build one or all Dexter services")
	}
	if config.IsCommandAvailable("status") {
		ui.PrintInfo("status     [service] Check the health of one or all services")
	}
	if config.IsCommandAvailable("start") {
		ui.PrintInfo("start      <service> Start a Dexter service")
	}
	if config.IsCommandAvailable("stop") {
		ui.PrintInfo("stop       <service> Stop a Dexter service")
	}
	if config.IsCommandAvailable("restart") {
		ui.PrintInfo("restart    <service> Restart a Dexter service")
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
		ui.PrintInfo("python     [<subcommand>] [args...] Manage Dexter's Python environment or run Python commands")
	}
	if config.IsCommandAvailable("bun") {
		ui.PrintInfo("bun        [args...] Proxy for the system's bun executable")
	}
	if config.IsCommandAvailable("bunx") {
		ui.PrintInfo("bunx       [args...] Proxy for the system's bunx executable")
	}
	if config.IsCommandAvailable("pull") {
		ui.PrintInfo("pull       Pull latest changes for all Dexter services")
	}
	ui.PrintInfo("version    Show version information")
	ui.PrintInfo("help       Show this help message")
	ui.PrintInfo("Dexter root:        ~/Dexter")
	ui.PrintInfo("EasterCompany root: ~/EasterCompany")
	fmt.Println()
}
