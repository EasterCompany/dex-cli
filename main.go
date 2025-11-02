package main

import (
	"fmt"
	"os"

	"github.com/EasterCompany/dex-cli/cmd"
	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/ui"
)

const (
	version = "1.0.0"
)

var (
	commit = "unknown"
	date   = "unknown"
)

func main() {
	isDevMode := isDevMode()
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
	case "pull":
		if err := cmd.Pull(); err != nil {
			ui.PrintError(fmt.Sprintf("Error: %v", err))
			os.Exit(1)
		}

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
		ui.PrintInfo(fmt.Sprintf("dex-cli v%s @ %s-%s", version, commit, date))

	case "build":
		if len(os.Args) < 3 {
			ui.PrintError("Error: service name or 'all' required for 'build' command")
			printUsage(isDevMode, hasSourceServices)
			os.Exit(1)
		}
		service := os.Args[2]
		if err := cmd.Build(service); err != nil {
			ui.PrintError(fmt.Sprintf("Error: %v", err))
			os.Exit(1)
		}

	case "watch":
		if err := cmd.Watch(); err != nil {
			ui.PrintError(fmt.Sprintf("Error: %v", err))
			os.Exit(1)
		}

	case "config":
		if len(os.Args) < 3 {
			ui.PrintError("Error: subcommand required for 'config' command")
			printUsage(isDevMode, hasSourceServices)
			os.Exit(1)
		}
		subcommand := os.Args[2]
		if err := cmd.Config(subcommand); err != nil {
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
		if len(os.Args) < 3 {
			ui.PrintError("Error: service name required for 'logs' command")
			printUsage(isDevMode, hasSourceServices)
			os.Exit(1)
		}
		service := os.Args[2]
		follow := len(os.Args) > 3 && os.Args[3] == "-f"
		if err := cmd.Logs(service, follow); err != nil {
			ui.PrintError(fmt.Sprintf("Error: %v", err))
			os.Exit(1)
		}

	case "model":
		if err := cmd.Model(os.Args[2:]); err != nil {
			if ui.PrintError(fmt.Sprintf("Error: %v", err)); err != nil {
				os.Exit(1)
			}
		}

	case "format":
		if hasSourceServices {
			if err := cmd.Format(os.Args[2:]); err != nil {
				if ui.PrintError(fmt.Sprintf("Error: %v", err)); err != nil {
					os.Exit(1)
				}
			}
		} else {
			ui.PrintError(fmt.Sprintf("Unknown command: %s", command))
			printUsage(isDevMode, hasSourceServices)
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

	case "lint":
		if hasSourceServices {
			if err := cmd.Lint(os.Args[2:]); err != nil {
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

func isDevMode() bool {
	// Check if the source code directory exists
	path, err := config.ExpandPath("~/EasterCompany/dex-cli")
	if err != nil {
		return false
	}
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}

func printUsage(isDevMode bool, hasSourceServices bool) {
	ui.PrintInfo("dex <command> [options]")
	ui.PrintInfo("pull       Clone/pull all Dexter services from Git")
	if isDevMode {
		ui.PrintInfo("update     Update dex-cli to latest version")
	}
	ui.PrintInfo("build      <service|all> Build one or all Dexter services")
	ui.PrintInfo("status     [service] Check the health of one or all services")
	ui.PrintInfo("start      <service> Start a Dexter service")
	ui.PrintInfo("stop       <service> Stop a Dexter service")
	ui.PrintInfo("restart    <service> Restart a Dexter service")
	ui.PrintInfo("config     <validate> Manage and validate configuration files")
	ui.PrintInfo("watch      Show a live dashboard of all service statuses")
	ui.PrintInfo("logs       <service> [-f] View service logs")
	ui.PrintInfo("model      <list|delete> Manage Dexter models")
	if hasSourceServices {
		ui.PrintInfo("format     Format and lint all code")
		ui.PrintInfo("lint       Lint all code")
		ui.PrintInfo("test       Run all tests")
	}
	ui.PrintInfo("system     Show system info and manage packages")
	ui.PrintInfo("version    Show version information")
	ui.PrintInfo("help       Show this help message")
	ui.PrintInfo("Dexter root:        ~/Dexter")
	ui.PrintInfo("EasterCompany root: ~/EasterCompany")
	fmt.Println()
}
