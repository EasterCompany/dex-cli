package cmd

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/ui"
)

func runOnAllServices(command string, args []string, showOutputOnFailure bool) error {
	logFile, err := config.LogFile()
	if err != nil {
		return fmt.Errorf("failed to get log file: %w", err)
	}
	defer func() { _ = logFile.Close() }()

	log := func(message string) {
		_, _ = fmt.Fprintln(logFile, message)
	}

	log(fmt.Sprintf("Running command '%s' on all services.", command))

	// Load the service map
	serviceMap, err := config.LoadServiceMap()
	if err != nil {
		return fmt.Errorf("failed to load service map: %w", err)
	}

	var services []config.ServiceEntry
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
			log(fmt.Sprintf("Error expanding path for service %s: %v", service.ID, err))
		} else {
			cmd := exec.Command(command, args...)
			cmd.Dir = path
			out, err := cmd.CombinedOutput()
			if err != nil {
				status = "FAILED"
				if showOutputOnFailure {
					fmt.Printf("▼ %s\n", service.ID)
					fmt.Println(string(out))
				}
				log(fmt.Sprintf("Command '%s' failed for service %s. Output:\n%s", command, service.ID, string(out)))
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
