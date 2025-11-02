package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	// Table styles
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("99")).
			Padding(0, 1)

	cellStyle = lipgloss.NewStyle().
			Padding(0, 1)

	statusHealthyStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("42")).
				Bold(true)

	statusDegradedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("220")).
				Bold(true)

	statusUnhealthyStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("196")).
				Bold(true)

	statusDownStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Bold(true)

	serviceNameStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("86"))
)

type TableColumn struct {
	Header string
	Width  int
}

type TableRow []string

type Table struct {
	Columns []TableColumn
	Rows    []TableRow
}

// RenderTable renders a table with proper alignment and styling
func RenderTable(table Table) string {
	var output strings.Builder

	// Render header
	headerRow := make([]string, len(table.Columns))
	for i, col := range table.Columns {
		headerRow[i] = padRight(col.Header, col.Width)
	}
	output.WriteString(headerStyle.Render(strings.Join(headerRow, " ")))
	output.WriteString("\n")

	// Render separator
	separator := make([]string, len(table.Columns))
	for i, col := range table.Columns {
		separator[i] = strings.Repeat("─", col.Width)
	}
	output.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(strings.Join(separator, " ")))
	output.WriteString("\n")

	// Render rows
	for _, row := range table.Rows {
		cells := make([]string, len(table.Columns))
		for i, cell := range row {
			if i < len(table.Columns) {
				cells[i] = padRight(cell, table.Columns[i].Width)
			}
		}
		output.WriteString(cellStyle.Render(strings.Join(cells, " ")))
		output.WriteString("\n")
	}

	return output.String()
}

// padRight pads a string to the right with spaces
func padRight(s string, width int) string {
	// Remove ANSI codes for length calculation
	visibleLen := lipgloss.Width(s)
	if visibleLen >= width {
		return s
	}
	return s + strings.Repeat(" ", width-visibleLen)
}

// StyleStatus applies color to status text
func StyleStatus(status string) string {
	upper := strings.ToUpper(status)
	switch upper {
	case "HEALTHY":
		return statusHealthyStyle.Render(upper)
	case "DEGRADED":
		return statusDegradedStyle.Render(upper)
	case "UNHEALTHY", "DOWN", "ERROR":
		return statusUnhealthyStyle.Render(upper)
	case "SKIPPED":
		return statusDownStyle.Render(upper)
	case "INVALID RESP":
		return statusUnhealthyStyle.Render("INVALID")
	default:
		return cellStyle.Render(upper)
	}
}

// StyleServiceName applies color to service names
func StyleServiceName(name string) string {
	return serviceNameStyle.Render(name)
}

// StyleVersion returns version text (no special styling)
func StyleVersion(version string) string {
	if version == "N/A" || version == "" {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("N/A")
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Render(version)
}

// StyleUptime returns uptime text
func StyleUptime(uptime string) string {
	if uptime == "N/A" || uptime == "" {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("N/A")
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color("111")).Render(uptime)
}

// StyleTimestamp returns timestamp text
func StyleTimestamp(timestamp string) string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render(timestamp)
}

// StyleAddress returns address text
func StyleAddress(address string) string {
	if address == "N/A" || address == "" {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("N/A")
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color("147")).Render(address)
}

// CreateServiceTable creates a properly formatted service status table
func CreateServiceTable(rows []TableRow) Table {
	return Table{
		Columns: []TableColumn{
			{Header: "SERVICE", Width: 25},
			{Header: "ADDRESS", Width: 25},
			{Header: "VERSION", Width: 12},
			{Header: "STATUS", Width: 15},
			{Header: "UPTIME", Width: 10},
			{Header: "LAST CHECK", Width: 10},
		},
		Rows: rows,
	}
}

// FormatTableRow formats a row with proper styling
func FormatTableRow(service, address, version, status, uptime, timestamp string) TableRow {
	return TableRow{
		StyleServiceName(service),
		StyleAddress(address),
		StyleVersion(version),
		StyleStatus(status),
		StyleUptime(uptime),
		StyleTimestamp(timestamp),
	}
}

// FormatFormatTableRow formats a row with proper styling for the format command
func FormatFormatTableRow(service, status string) TableRow {
	return TableRow{
		StyleServiceName(service),
		StyleStatus(status),
	}
}

// PrintLogo prints the Dexter ASCII logo
func PrintLogo() {
	logo := lipgloss.NewStyle().
		Foreground(lipgloss.Color("99")).
		Bold(true).
		Render(`
 ██████╗ ███████╗██╗  ██╗████████╗███████╗██████╗
 ██╔══██╗██╔════╝╚██╗██╔╝╚══██╔══╝██╔════╝██╔══██╗
 ██║  ██║█████╗   ╚███╔╝    ██║   █████╗  ██████╔╝
 ██║  ██║██╔══╝   ██╔██╗    ██║   ██╔══╝  ██╔══██╗
 ██████╔╝███████╗██╔╝ ██╗   ██║   ███████╗██║  ██║
 ╚═════╝ ╚══════╝╚═╝  ╚═╝   ╚═╝   ╚══════╝╚═╝  ╚═╝
`)
	fmt.Println(logo)
}
