package ui

import (
	"fmt"
	"strings"
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

func NewTable(headers []string) Table {
	columns := make([]TableColumn, len(headers))
	for i, header := range headers {
		columns[i] = TableColumn{Header: header, Width: len(header)}
	}
	return Table{Columns: columns}
}

func (t *Table) AddRow(row TableRow) {
	t.Rows = append(t.Rows, row)
	for i, cell := range row {
		// Calculate width based on the visible string, not the raw one with ANSI codes
		visibleCell := StripANSI(cell)
		if len(visibleCell) > t.Columns[i].Width {
			t.Columns[i].Width = len(visibleCell)
		}
	}
}

// Truncate shortens a string to a max length, adding an ellipsis if necessary.
func Truncate(s string, maxLength int) string {
	if StripANSI(s) == s { // Only truncate if no ANSI codes
		if len(s) > maxLength {
			return s[:maxLength-3] + "..."
		}
	}
	return s // Return colorized strings as-is
}

func (t *Table) Render() {
	// Header
	header := ""
	for _, col := range t.Columns {
		header += padRight(col.Header, col.Width) + " "
	}
	fmt.Println(header)

	// Separator
	separator := ""
	for _, col := range t.Columns {
		separator += strings.Repeat("-", col.Width) + " "
	}
	fmt.Println(separator)

	// Rows
	for _, row := range t.Rows {
		rowStr := ""
		for i, cell := range row {
			rowStr += padRight(cell, t.Columns[i].Width) + " "
		}
		fmt.Println(rowStr)
	}
}

func padRight(s string, width int) string {
	// Get the visible length of the string, stripping ANSI codes
	visibleLength := len(StripANSI(s))

	if visibleLength >= width {
		return s
	}

	// Calculate padding based on the visible length
	return s + strings.Repeat(" ", width-visibleLength)
}

func FormatFormatTableRow(service, status string) TableRow {
	return TableRow{service, status}
}

func RenderTable(table Table) string {
	var output strings.Builder

	// Render header
	headerRow := make([]string, len(table.Columns))
	for i, col := range table.Columns {
		headerRow[i] = padRight(col.Header, col.Width)
	}
	output.WriteString(strings.Join(headerRow, " "))
	output.WriteString("\n")

	// Render separator
	separator := make([]string, len(table.Columns))
	for i, col := range table.Columns {
		separator[i] = strings.Repeat("-", col.Width)
	}
	output.WriteString(strings.Join(separator, " "))
	output.WriteString("\n")

	// Render rows
	for _, row := range table.Rows {
		cells := make([]string, len(table.Columns))
		for i, cell := range row {
			if i < len(table.Columns) {
				cells[i] = padRight(cell, table.Columns[i].Width)
			}
		}
		output.WriteString(strings.Join(cells, " "))
		output.WriteString("\n")
	}

	return output.String()
}

func CreateServiceTable(rows []TableRow) Table {
	table := NewTable([]string{"SERVICE", "ADDRESS", "VERSION", "STATUS", "UPTIME", "GOROUTINES", "MEM (MB)", "LAST CHECK"})
	for _, row := range rows {
		table.AddRow(row)
	}
	return table
}

func FormatTableRow(service, address, version, status, uptime, goroutines, mem, timestamp string) TableRow {
	return TableRow{service, address, version, status, uptime, goroutines, mem, timestamp}
}
