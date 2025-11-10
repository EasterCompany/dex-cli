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
		if i < len(t.Columns) {
			// Calculate width based on the visible string, not the raw one with ANSI codes
			visibleCell := StripANSI(cell)
			if len(visibleCell) > t.Columns[i].Width {
				t.Columns[i].Width = len(visibleCell)
			}
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
	var output strings.Builder

	// Render header
	headerRow := make([]string, len(t.Columns))
	for i, col := range t.Columns {
		// Apply Cyan color to headers
		headerRow[i] = fmt.Sprintf("%s%s%s", ColorCyan, padRight(col.Header, col.Width), ColorReset)
	}
	output.WriteString(strings.Join(headerRow, "  ")) // Increased spacing
	output.WriteString("\n")

	// Render separator
	separator := make([]string, len(t.Columns))
	for i, col := range t.Columns {
		// Use Dark Gray color for separators
		separator[i] = fmt.Sprintf("%s%s%s", ColorDarkGray, strings.Repeat(BorderHorizontal, col.Width), ColorReset)
	}
	output.WriteString(strings.Join(separator, "  ")) // Increased spacing
	output.WriteString("\n")

	// Render rows
	for _, row := range t.Rows {
		cells := make([]string, len(t.Columns))
		for i, cell := range row {
			if i < len(t.Columns) {
				// Default row color is White/Reset
				cells[i] = padRight(cell, t.Columns[i].Width)
			}
		}
		output.WriteString(strings.Join(cells, "  ")) // Increased spacing
		output.WriteString("\n")
	}

	PrintRaw(output.String())
}

// padRight pads a string with spaces up to the specified width.
func padRight(s string, width int) string {
	visibleLength := len(StripANSI(s))
	// Calculate padding based on the visible length
	return s + strings.Repeat(" ", width-visibleLength)
}

func FormatFormatTableRow(service, status string) TableRow {
	return TableRow{service, status}
}

func CreateServiceTable(rows []TableRow) Table {
	// Adjusted columns for a more modern, compact CLI look
	table := NewTable([]string{"SERVICE", "ADDRESS", "VERSION", "BRANCH", "COMMIT", "STATUS", "UPTIME", "SOURCE"})
	for _, row := range rows {
		table.AddRow(row)
	}
	return table
}

// PrintVersionComparison prints a comparison of the old and new versions.
func PrintVersionComparison(currentVersionStr, newVersionStr, latestVersion, buildYear string, currentSize, newSize, oldSize, newSize2 int64) {
	fmt.Printf("Version updated from %s to %s\n", currentVersionStr, newVersionStr)
}
