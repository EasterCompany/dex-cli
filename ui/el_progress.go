package ui

import (
	"fmt"
	"strings"
)

func PrintProgressBar(label string, current int) {
	if current < 0 {
		current = 0
	}

	if current > 100 {
		current = 100
	}

	filledBlocks := int((float64(current) / 100.0) * float64(ProgressBarWidth))
	emptyBlocks := ProgressBarWidth - filledBlocks

	bar := strings.Repeat(ProgressBarFilled, filledBlocks)
	bar += strings.Repeat(ProgressBarEmpty, emptyBlocks)

	barColor := ColorYellow
	if current == 100 {
		barColor = ColorGreen
	} else if current < 25 {
		barColor = ColorRed
	}

	// Use \r to stay on one line, clear to end of line with spaces
	output := fmt.Sprintf("\r%s%s:%s %s%s%s %s[%d%%]%s",
		ColorCyan,
		label,
		ColorReset,
		barColor,
		bar,
		ColorReset,
		ColorBlue,
		current,
		ColorReset,
	)

	PrintRaw(output)

	// Print newline only when complete
	if current == 100 {
		PrintRaw("\n")
	}
}

// PrintSpinner shows a simple spinner animation on one line
func PrintSpinner(label string, frame int) {
	spinChars := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	spinChar := spinChars[frame%len(spinChars)]

	output := fmt.Sprintf("\r%s%s%s %s%s",
		ColorCyan,
		spinChar,
		ColorReset,
		label,
		ColorReset,
	)

	PrintRaw(output)
}

// ClearLine clears the current line
func ClearLine() {
	PrintRaw("\r\033[K")
}
