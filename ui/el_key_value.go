package ui

import (
	"fmt"
	"strings"
)

// KeyValue represents a single key-value pair for the block.
type KeyValue struct {
	Key   string
	Value string
}

// PrintKeyValBlock renders a clean, boxed configuration block.
func PrintKeyValBlock(title string, data []KeyValue) {
	// Find the maximum key width for alignment
	maxKeyLen := len(TitleCase(title))
	for _, kv := range data {
		if len(StripANSI(kv.Key)) > maxKeyLen {
			// Use StripANSI to get the visible length of the key
			keyVisibleLen := len(StripANSI(kv.Key))
			if keyVisibleLen > maxKeyLen {
				maxKeyLen = keyVisibleLen
			}
		}
	}

	// Add padding for aesthetics
	maxKeyLen += 2

	var sb strings.Builder

	// Top Border (Title Case and Purple)
	titleStyled := TitleCase(title)

	// Determine the length of the top bar required
	barLength := maxKeyLen + 1 + len(titleStyled) + 2 // Key-val width + space + title length + extra border space

	sb.WriteString(fmt.Sprintf("%s%s%s%s %s %s%s\n", ColorDarkGray, BorderTopLeft, strings.Repeat(BorderHorizontal, 3), ColorReset, ColorPurple, titleStyled, ColorReset))

	// Data Rows
	for _, kv := range data {
		// Key: Cyan, Value: Green
		keyVisibleLen := len(StripANSI(kv.Key))
		keyPadded := kv.Key + strings.Repeat(" ", maxKeyLen-keyVisibleLen)

		sb.WriteString(fmt.Sprintf("%s%s %s%s%s %s%s%s\n",
			ColorDarkGray, BorderVertical,
			ColorCyan, keyPadded, ColorReset,
			ColorGreen, kv.Value, ColorReset,
		))
	}

	// Bottom Border - Match the width of the top border
	sb.WriteString(fmt.Sprintf("%s%s%s%s\n", ColorDarkGray, BorderBottomLeft, strings.Repeat(BorderHorizontal, barLength+20), BorderBottomRight)) // Use a long enough line for the demo

	PrintRaw(sb.String())
}
