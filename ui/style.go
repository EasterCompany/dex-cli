package ui

import "fmt"

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
