package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

const (
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorWhite  = "\033[37m"
	ColorReset  = "\033[0m"
)

func PrintTitle(title string) {
	fmt.Printf("%s=== %s ===%s\n", ColorCyan, title, ColorReset)
}

func PrintSectionTitle(title string) {
	fmt.Printf("\n%s--- %s ---%s\n", ColorPurple, title, ColorReset)
}

func PrintSuccess(message string) {
	fmt.Printf("%s✓ %s%s\n", ColorGreen, message, ColorReset)
}

func PrintError(message string) {
	fmt.Printf("%s✗ %s%s\n", ColorRed, message, ColorReset)
}

func PrintWarning(message string) {
	fmt.Printf("%s⚠ %s%s\n", ColorYellow, message, ColorReset)
}

func PrintInfo(message string) {
	fmt.Printf("%s- %s%s\n", ColorBlue, message, ColorReset)
}

// Colorize wraps a string with the given ANSI color codes
func Colorize(text string, color string) string {
	return fmt.Sprintf("%s%s%s", color, text, ColorReset)
}

func RenderTitle(title string) string {
	style := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("63")).
		Padding(1, 2)

	return style.Render(title)
}

func RenderSubtitle(subtitle string) string {
	style := lipgloss.NewStyle().
		Italic(true).
		Foreground(lipgloss.Color("245"))

	return style.Render(subtitle)
}

func RenderSectionTitle(title string) string {
	style := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("99")).
		Padding(0, 1).
		MarginTop(1)

	return style.Render(title)
}

func PrintSection(title string) {
	fmt.Printf("\n%s=== %s ===%s\n", ColorCyan, title, ColorReset)
}

func PrintVersionComparison(oldVersion, newVersion string, oldSize, newSize int64, additions, deletions int) {
	fmt.Printf("%s  Previous version: %s%s\n", ColorBlue, ColorReset, oldVersion)
	fmt.Printf("%s  Current version:  %s%s\n", ColorBlue, ColorReset, newVersion)

	// Calculate size difference
	sizeDiff := newSize - oldSize
	var sizeColor string
	var sizeIndicator string

	if sizeDiff > 0 {
		sizeColor = ColorRed
		sizeIndicator = "↑"
	} else if sizeDiff < 0 {
		sizeColor = ColorGreen
		sizeIndicator = "↓"
	} else {
		sizeColor = ColorYellow
		sizeIndicator = "="
	}

	// Format sizes
	oldSizeStr := formatBytes(oldSize)
	newSizeStr := formatBytes(newSize)
	diffSizeStr := formatBytes(abs(sizeDiff))

	fmt.Printf("%s  Binary size:      %s%s → %s%s(%s %s)%s\n",
		ColorBlue, ColorReset, oldSizeStr, newSizeStr,
		sizeColor, sizeIndicator, diffSizeStr, ColorReset)

	// Show source changes if available
	if additions > 0 || deletions > 0 {
		fmt.Printf("%s  Source changes:   %s%s+%d%s %s-%d%s\n",
			ColorBlue, ColorReset,
			ColorGreen, additions, ColorReset,
			ColorRed, deletions, ColorReset)
	}
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func abs(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}
