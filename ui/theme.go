package ui

import (
	"fmt"
	"regexp"
	"strings"
)

// --- Color and ANSI Constants ---

const (
	ColorRed       = `\033[31m`
	ColorBrightRed = `\033[91m` // Used for SubHeaders (Orange-like)
	ColorGreen     = `\033[32m`
	ColorYellow    = `\033[33m` // Used for Headers
	ColorBlue      = `\033[34m`
	ColorPurple    = `\033[35m`
	ColorCyan      = `\033[36m`
	ColorWhite     = `\033[37m`
	ColorDarkGray  = `\033[90m`
	ColorReset     = `\033[0m`
)

// Global UI Constants and Emojis
const (
	// Border elements
	BorderVertical    = "│"
	BorderHorizontal  = "─"
	BorderTopLeft     = "╭"
	BorderTopRight    = "╮"
	BorderBottomLeft  = "╰"
	BorderBottomRight = "╯"

	// Status Emojis
	EmojiSuccess = "✅"
	EmojiFailure = "❌"
	EmojiRunning = "⚙️"
	EmojiWaiting = "⏳"
	EmojiInfo    = "💡"

	// Progress Bar
	ProgressBarFilled = "█" // Full block
	ProgressBarEmpty  = "░" // Light shade
	ProgressBarWidth  = 40
)

// ansiRegex is used to strip ANSI escape codes.
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// StripANSI removes ANSI color codes from a string.
func StripANSI(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}

// PrintRaw is the lowest-level printing function, used by all other functions.
func PrintRaw(s string) {
	fmt.Print(s)
}

// --- Custom Header and Sub-Header Logic ---

// TitleCase capitalizes the first letter of every word in a string.
func TitleCase(s string) string {
	s = strings.ToLower(s)
	words := strings.Fields(s)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + word[1:]
		}
	}
	return strings.Join(words, " ")
}

// PrintHeader prints a primary header, enforcing YELLOW text and UPPERCASE.
// Example: # TEXT-GOES-HERE
func PrintHeader(title string) {
	styledTitle := strings.ToUpper(title)
	// Output: Yellow Block + Title + Yellow Block
	PrintRaw(fmt.Sprintf("\n%s%s %s %s%s\n", ColorYellow, "█", styledTitle, "█", ColorReset))
	// Add a subtle separator line
	PrintRaw(fmt.Sprintf("%s%s%s\n", ColorDarkGray, strings.Repeat(BorderHorizontal, 80), ColorReset))
}

// PrintSubHeader prints a secondary header, enforcing BRIGHT RED/ORANGE text and Capitalization (Title Case).
// Example: ## Text-Goes-Here
func PrintSubHeader(title string) {
	styledTitle := TitleCase(title)
	// Output: Bright Red/Orange Arrow + Title
	PrintRaw(fmt.Sprintf("\n%s▶ %s%s\n", ColorBrightRed, styledTitle, ColorReset))
}

// --- Basic Utility Printing (Used in main.go) ---

// PrintInfo prints a standard informational message.
func PrintInfo(message string) {
	PrintRaw(fmt.Sprintf("%s%s %s%s\n", ColorCyan, "i", message, ColorReset))
}

// PrintSuccess prints a standard success message.
func PrintSuccess(message string) {
	PrintRaw(fmt.Sprintf("%s✓ %s%s\n", ColorGreen, message, ColorReset))
}

// PrintError prints a standard error message.
func PrintError(message string) {
	PrintRaw(fmt.Sprintf("%s✕ %s%s\n", ColorRed, message, ColorReset))
}

// PrintWarning prints a standard warning message.
func PrintWarning(message string) {
	PrintRaw(fmt.Sprintf("%s! %s%s\n", ColorYellow, message, ColorReset))
}
