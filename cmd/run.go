package cmd

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/ui"
)

type commandResult struct {
	service string
	status  string
}

func runOnAllServices(command string, args []string, title string, showOutputOnFailure bool) error {
	fmt.Println(ui.RenderTitle(title))

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
		services = append(services, s...)
	}

	var rows []ui.TableRow
	for _, service := range services {
		if service.ID != "dex-cli" && !strings.HasPrefix(service.ID, "dex-") {
			continue
		}
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
