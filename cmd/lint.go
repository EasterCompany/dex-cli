package cmd

import (
	"fmt"
	"os/exec"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/ui"
)

// Lint lints the code for all services
func Lint(args []string) error {
	fmt.Println(ui.RenderTitle("LINTING"))

	// Load the service map
	serviceMap, err := config.LoadServiceMap()
	if err != nil {
		return fmt.Errorf("failed to load service map: %w", err)
	}

	var rows []ui.TableRow
	for _, services := range serviceMap.Services {
		for _, service := range services {
			status := "LINTED"
			path, err := config.ExpandPath(service.Source)
			if err != nil {
				status = "ERROR"
			} else {
				if err := lintGoFiles(path); err != nil {
					status = "ERROR"
				}
			}
			rows = append(rows, ui.FormatFormatTableRow(service.ID, status))
		}
	}

	// Render table
	table := createLintTable(rows)
	fmt.Print(ui.RenderTable(table))

	return nil
}

func lintGoFiles(path string) error {
	cmd := exec.Command("golint", "-set_exit_status", "./...")
	cmd.Dir = path
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(string(out))
		return fmt.Errorf("failed to lint Go files: %w", err)
	}
	return nil
}

func createLintTable(rows []ui.TableRow) ui.Table {
	return ui.Table{
		Columns: []ui.TableColumn{
			{Header: "SERVICE", Width: 25},
			{Header: "STATUS", Width: 15},
		},
		Rows: rows,
	}
}
