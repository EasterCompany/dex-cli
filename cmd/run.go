package cmd

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/ui"
)

func runOnAllServices(command string, args []string, showOutputOnFailure bool) error {

	// Load the service map
	serviceMap, err := config.LoadServiceMap()
	if err != nil {
		return fmt.Errorf("failed to load service map: %w", err)
	}

	// Add dex-cli to the list of services
	services := []config.ServiceEntry{{
		ID:     "dex-cli",
		Source: ".",
	}}
	for _, s := range serviceMap.Services {
		for _, service := range s {
			if strings.HasPrefix(service.ID, "dex-") && service.Source != "" {
				services = append(services, service)
			}
		}
	}

	var rows []ui.TableRow
	for _, service := range services {
		status := "PASSED"
		path, err := config.ExpandPath(service.Source)
		if err != nil {
			status = "ERROR"
		} else {
			cmd := exec.Command(command, args...)
			cmd.Dir = path
			out, err := cmd.CombinedOutput()
			if err != nil {
				status = "FAILED"
				fmt.Println(ui.RenderSubtitle(fmt.Sprintf("▼ %s", service.ID)))
				fmt.Println(string(out))
			}
		}
		rows = append(rows, ui.FormatFormatTableRow(service.ID, status))
	}

	// Render table
	table := createTable(rows)
	fmt.Print(ui.RenderTable(table))

	return nil
}

func createTable(rows []ui.TableRow) ui.Table {
	return ui.Table{
		Columns: []ui.TableColumn{
			{Header: "SERVICE", Width: 30},
			{Header: "STATUS", Width: 10},
		},
		Rows: rows,
	}
}
