package ui

import (
	"fmt"
	"strings"
)

type TableColumn struct {
	Header   string
	Width    int
	MaxWidth int // 0 means no max width
	MinWidth int // 0 means use header length as minimum
}

type TableRow []string

type Table struct {
	Columns []TableColumn
	Rows    []TableRow
}

func NewTable(headers []string) Table {
	columns := make([]TableColumn, len(headers))
	for i, header := range headers {
		columns[i] = TableColumn{Header: header, Width: len(header), MaxWidth: 0, MinWidth: 0}
	}
	return Table{Columns: columns}
}

// NewTableWithWidths creates a table with specified column widths.
// Use 0 for maxWidth to allow unlimited width, or specify a max width to truncate with "..."
func NewTableWithWidths(headers []string, maxWidths []int) Table {
	columns := make([]TableColumn, len(headers))
	for i, header := range headers {
		maxW := 0
		if i < len(maxWidths) {
			maxW = maxWidths[i]
		}
		columns[i] = TableColumn{
			Header:   header,
			Width:    len(header),
			MaxWidth: maxW,
			MinWidth: len(header),
		}
	}
	return Table{Columns: columns}
}

func (t *Table) AddRow(row TableRow) {
	t.Rows = append(t.Rows, row)
	for i, cell := range row {
		if i < len(t.Columns) {
			// Calculate width based on visible characters (counting runes, not bytes)
			visibleLen := VisibleLength(cell)

			// Apply max width constraint
			if t.Columns[i].MaxWidth > 0 && visibleLen > t.Columns[i].MaxWidth {
				visibleLen = t.Columns[i].MaxWidth
			}

			// Update width if larger (but respect max width)
			if visibleLen > t.Columns[i].Width {
				t.Columns[i].Width = visibleLen
			}

			// Ensure minimum width
			if t.Columns[i].MinWidth > 0 && t.Columns[i].Width < t.Columns[i].MinWidth {
				t.Columns[i].Width = t.Columns[i].MinWidth
			}
		}
	}
}

// Truncate shortens a string to a max length, adding an ellipsis if necessary.
// Handles ANSI color codes and Unicode characters properly.
func Truncate(s string, maxLength int) string {
	if maxLength <= 0 {
		return s
	}

	visibleLen := VisibleLength(s)
	if visibleLen <= maxLength {
		return s
	}

	stripped := StripANSI(s)

	// If the string has no ANSI codes, simple rune-based truncation
	if stripped == s {
		runes := []rune(s)
		if maxLength <= 3 {
			return string(runes[:maxLength])
		}
		return string(runes[:maxLength-3]) + "..."
	}

	// For strings with ANSI codes, we need to preserve color codes
	// Extract visible characters up to maxLength-3, then add "..."
	visibleCount := 0
	targetVisible := maxLength - 3
	if maxLength <= 3 {
		targetVisible = maxLength
	}

	var result strings.Builder
	inAnsi := false
	runes := []rune(s)

	for i := 0; i < len(runes); i++ {
		if runes[i] == '\x1b' {
			inAnsi = true
		}

		if inAnsi {
			result.WriteRune(runes[i])
			if runes[i] == 'm' {
				inAnsi = false
			}
		} else {
			if visibleCount >= targetVisible {
				break
			}
			result.WriteRune(runes[i])
			visibleCount++
		}
	}

	if maxLength > 3 {
		result.WriteString("...")
	}

	// Add reset to ensure colors don't bleed
	result.WriteString(ColorReset)

	return result.String()
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
				// Truncate cell if it exceeds max width
				truncatedCell := cell
				if t.Columns[i].MaxWidth > 0 {
					truncatedCell = Truncate(cell, t.Columns[i].MaxWidth)
				}
				// Pad to column width
				cells[i] = padRight(truncatedCell, t.Columns[i].Width)
			}
		}
		output.WriteString(strings.Join(cells, "  ")) // Increased spacing
		output.WriteString("\n")
	}

	PrintRaw(output.String())
}

// padRight pads a string with spaces up to the specified width.
// Uses rune count for proper Unicode character width calculation.
func padRight(s string, width int) string {
	visibleLength := VisibleLength(s)
	padding := width - visibleLength
	if padding < 0 {
		padding = 0
	}
	return s + strings.Repeat(" ", padding)
}

func FormatFormatTableRow(service, status string) TableRow {
	return TableRow{service, status}
}

func CreateServiceTable(rows []TableRow) Table {
	// Adjusted columns for a more modern, compact CLI look
	table := NewTable([]string{"SERVICE", "ADDRESS", "VERSION", "BRANCH", "COMMIT", "STATUS", "UPTIME"})
	for _, row := range rows {
		table.AddRow(row)
	}
	return table
}

// PrintVersionComparison prints a comparison of the old and new versions.
func PrintVersionComparison(currentVersionStr, newVersionStr, latestVersion, buildYear string, currentSize, newSize, oldSize, newSize2 int64) {
	fmt.Printf("Version updated from %s to %s\n", currentVersionStr, newVersionStr)
}
