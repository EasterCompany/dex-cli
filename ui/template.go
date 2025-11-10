package ui

import (
	"fmt"
)

// PrintTemplateDemo shows off all the new UI elements for development and testing.
func PrintTemplateDemo() {
	// --- Header Demonstration ---
	PrintHeader("CLI UI MODEL DEMONSTRATION")

	PrintSubHeader("Configuration and Project Status")

	// --- Key Value Block 1 (Standard) ---
	configData := []KeyVal{
		{Key: "Project Name", Value: "Aurora"},
		{Key: "Version", Value: "v2.1.0-beta"},
		{Key: "Environment", Value: "Production"},
		{Key: "API Endpoint", Value: "https://api.aurora.com/v1"},
	}
	PrintKeyValBlock("Project Details", configData)

	// --- Key Value Block 2 (With Colorized Keys) ---
	PrintRaw("\n") // Add separation
	environmentData := []KeyVal{
		{Key: fmt.Sprintf("%sDEBUG%s Mode", ColorYellow, ColorReset), Value: "true"},
		{Key: fmt.Sprintf("%sMax Threads%s", ColorPurple, ColorReset), Value: "16"},
		{Key: fmt.Sprintf("%sStatus%s", ColorGreen, ColorReset), Value: "Online"},
		{Key: fmt.Sprintf("%sLicense Key%s (Redacted)", ColorDarkGray, ColorReset), Value: "****-****-****-1234"},
	}
	PrintKeyValBlock("Environment Settings", environmentData)

	// --- Status Indicators ---
	PrintSubHeader("Component Health Check")

	PrintSuccessStatus("Database connection verified.")
	PrintRunningStatus("Worker process 'task-queue' is active.")
	PrintFailureStatus("Authentication service failed to start. (Error 503)")
	PrintInfoStatus("Please update configuration file for new features.")
	PrintStatusIndicator("wait", "Waiting for network handshake...")

	// --- Progress Bar Demonstration ---
	PrintSubHeader("Long-Running Operations")

	PrintRaw("\n")

	// Progress 1
	PrintProgressBar("Fetching Data", 15)

	// Progress 2
	PrintProgressBar("Processing Assets", 68)

	// Progress 3 (Completed)
	PrintProgressBar("Deployment Complete", 100)

	// --- Table Example ---
	PrintSubHeader("Service Status Table")

	serviceRows := []TableRow{
		{"auth-api", "127.0.0.1:8080", "1.5.0", fmt.Sprintf("%s%s%s", ColorGreen, "Running", ColorReset), "2h 45m"},
		{"db-worker", "localhost", "2.0.1", fmt.Sprintf("%s%s%s", ColorRed, "Failed", ColorReset), "5s"},
		{"task-queue", "10.0.0.5:9000", "1.0.0", fmt.Sprintf("%s%s%s", ColorYellow, "Pending", ColorReset), "N/A"},
	}

	// FIX: Assign the table to a variable before calling the pointer method Render().
	serviceTable := CreateServiceTable(serviceRows)
	serviceTable.Render()

	// --- Code Block Example: Go ---
	PrintSubHeader("Source Code Snippet (Go)")

	goCode := `package main

import (
	"fmt"
	"os"
)

// The main entry point for the CLI.
func func() {
	if len(os.Args) < 2 {
		fmt.Println("Error: command required")
		return
	}
	// process command here
	fmt.Println("Command executed successfully.")
}
`
	PrintCodeBlock(CodeSnippet{
		FileName:    "main.go",
		SizeKB:      16.5,
		Status:      "2 warnings",
		CodeContent: goCode,
		Language:    "go",
	})

	// --- Code Block Example: Python ---
	PrintSubHeader("Source Code Snippet (Python)")

	pythonCode := `# Initialize a user class
class User:
    def __init__(self, name, active):
        self.name = name
        self.active = active

    def greet(self):
        if self.active is True:
            return "Hello, " + self.name
        return None

user = User("Dexter", True)
print(user.greet())
`
	PrintCodeBlock(CodeSnippet{
		FileName:    "user.py",
		SizeKB:      0.8,
		Status:      "no issues",
		CodeContent: pythonCode,
		Language:    "python",
	})

	// --- Code Block Example: Markdown ---
	PrintSubHeader("Source Document Snippet (Markdown)")

	markdownContent := `# Project Documentation
	
## Installation
	
1. Clone the repository.
2. Run the build script.

> This is a blockquote about notes.

Check out the **latest** changes in ` + "`README.md`" + `.
*Important*: See the [Source Code](https://github.com/project).
`
	PrintCodeBlock(CodeSnippet{
		FileName:    "README.md",
		SizeKB:      1.2,
		Status:      "up-to-date",
		CodeContent: markdownContent,
		Language:    "markdown",
	})

	// --- Code Block Example: JSON (using the new helper) ---
	PrintSubHeader("Raw Configuration Output (JSON)")

	jsonBytes := []byte(`{"debug": true, "retries": 5, "cache_size_mb": 1024.5, "services": [{"name": "Auth", "enabled": true}, {"name": "Jobs", "enabled": false}], "token": null, "description": "This is a test string."}`)

	// Use the new helper function that handles pretty-printing and highlighting
	PrintCodeBlockFromBytes(jsonBytes, "config.json", "json")

	PrintHeader("TESTING COMPLETE")
}
