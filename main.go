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
	isDevMode := config.IsDevMode()
	hasSourceServices := config.HasSourceServices()

	if err := config.EnsureDirectoryStructure(); err != nil {
		ui.PrintError(fmt.Sprintf("Error ensuring directory structure: %v", err))
		os.Exit(1)
	}

	if len(os.Args) < 2 {
		printUsage(isDevMode, hasSourceServices)
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "update":
		if isDevMode {
			if err := cmd.Update(os.Args[2:]); err != nil {
				ui.PrintError(fmt.Sprintf("Error: %v", err))
				os.Exit(1)
			}
		} else {
			ui.PrintError(fmt.Sprintf("Unknown command: %s", command))
			printUsage(isDevMode, hasSourceServices)
			os.Exit(1)
		}

	case "system":
		if err := cmd.System(os.Args[2:]); err != nil {
			ui.PrintError(fmt.Sprintf("Error: %v", err))
			os.Exit(1)
		}

	case "version", "-v", "--version":
		cmd.Version(version, branch, commit, buildDate, buildYear)

	case "build":
		if err := cmd.Build(os.Args[2:]); err != nil {
			ui.PrintError(fmt.Sprintf("Error: %v", err))
			os.Exit(1)
		}

	case "start", "stop", "restart":
		if len(os.Args) < 3 {
			ui.PrintError(fmt.Sprintf("Error: service name required for '%s' command", command))
			printUsage(isDevMode, hasSourceServices)
			os.Exit(1)
		}
		service := os.Args[2]
		if err := cmd.Service(command, service); err != nil {
			ui.PrintError(fmt.Sprintf("Error: %v", err))
			os.Exit(1)
		}

	case "status":
		service := "all"
		if len(os.Args) > 2 {
			service = os.Args[2]
		}
		if err := cmd.Status(service); err != nil {
			ui.PrintError(fmt.Sprintf("Error: %v", err))
			os.Exit(1)
		}

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
		if err := cmd.Logs(args, follow); err != nil {
			ui.PrintError(fmt.Sprintf("Error: %v", err))
			os.Exit(1)
		}

	case "test":
		if hasSourceServices {
			if err := cmd.Test(os.Args[2:]); err != nil {
				if ui.PrintError(fmt.Sprintf("Error: %v", err)); err != nil {
					os.Exit(1)
				}
			}
		} else {
			ui.PrintError(fmt.Sprintf("Unknown command: %s", command))
			printUsage(isDevMode, hasSourceServices)
			os.Exit(1)
		}

	case "help", "-h", "--help":
		printUsage(isDevMode, hasSourceServices)

	default:
		ui.PrintError(fmt.Sprintf("Unknown command: %s", command))
		printUsage(isDevMode, hasSourceServices)
		os.Exit(1)
	}
}

func printUsage(isDevMode bool, hasSourceServices bool) {
	ui.PrintInfo("dex <command> [options]")
	if isDevMode {
		ui.PrintInfo("update     Update dex-cli to latest version")
	}
	ui.PrintInfo("build      <service|all> Build one or all Dexter services")
	ui.PrintInfo("status     [service] Check the health of one or all services")
	ui.PrintInfo("start      <service> Start a Dexter service")
	ui.PrintInfo("stop       <service> Stop a Dexter service")
	ui.PrintInfo("restart    <service> Restart a Dexter service")
	ui.PrintInfo("logs       <service> [-f] View service logs")
	if hasSourceServices {
		ui.PrintInfo("test       Run all tests")
	}
	ui.PrintInfo("system     Show system info and manage packages")
	ui.PrintInfo("version    Show version information")
	ui.PrintInfo("help       Show this help message")
	ui.PrintInfo("Dexter root:        ~/Dexter")
	ui.PrintInfo("EasterCompany root: ~/EasterCompany")
	fmt.Println()
}
