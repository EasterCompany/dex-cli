package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/ui"
)

// Format formats and lints the code for all services
func Format(args []string) error {
	fmt.Println(ui.RenderTitle("FORMATTING & LINTING"))

	// Load the service map
	serviceMap, err := config.LoadServiceMap()
	if err != nil {
		return fmt.Errorf("failed to load service map: %w", err)
	}

	var rows []ui.TableRow
	for _, services := range serviceMap.Services {
		for _, service := range services {
			status := "FORMATED"
			path, err := config.ExpandPath(service.Source)
			if err != nil {
				status = "ERROR"
			} else {
				if err := formatGoFiles(path); err != nil {
					status = "ERROR"
				}
			}
			rows = append(rows, ui.FormatFormatTableRow(service.ID, status))
		}
	}

	// Render table
	table := createFormatTable(rows)
	fmt.Print(ui.RenderTable(table))

	return nil
}

func formatGoFiles(path string) error {
	cmd := exec.Command("gofmt", "-w", path)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to format Go files: %w", err)
	}
	return nil
}

func createFormatTable(rows []ui.TableRow) ui.Table {
	return ui.Table{
		Columns: []ui.TableColumn{
			{Header: "SERVICE", Width: 25},
			{Header: "STATUS", Width: 15},
		},
		Rows: rows,
	}
}
