package main

import (
	"fmt"
	"os"

	"github.com/EasterCompany/dex-cli/cmd"
)

const (
	version = "1.0.0"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "pull":
		if err := cmd.Pull(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "system":
		if err := cmd.System(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "version", "-v", "--version":
		fmt.Printf("dex version %s\n", version)

	case "help", "-h", "--help":
		printUsage()

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Dexter CLI - Manage Dexter services")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  dex <command> [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  pull       Clone/pull all Dexter services from Git")
	fmt.Println("  system     Show system info and manage packages")
	fmt.Println("  version    Show version information")
	fmt.Println("  help       Show this help message")
	fmt.Println()
	fmt.Println("Environment:")
	fmt.Println("  Dexter root:        ~/Dexter")
	fmt.Println("  EasterCompany root: ~/EasterCompany")
	fmt.Println()
}
