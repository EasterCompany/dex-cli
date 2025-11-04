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

	output := fmt.Sprintf("%s%s:%s %s%s%s %s[%d%%]%s\n",
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
}
