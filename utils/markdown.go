package utils

import (
	"regexp"
	"strings"
)

// StandardizeReport handles formatting for reports intended for Discord and the Dashboard.
// It ensures double-newline spacing and converts headers to a 'safe' bold format
// that doesn't interfere with the backend report parser.
func StandardizeReport(input string) string {
	if input == "" {
		return ""
	}

	// 1. Convert '#' headers to **BOLD UPPERCASE**
	// This prevents the backend parser from splitting the report prematurely.
	reHeaders := regexp.MustCompile(`(?m)^#+\s+(.*)$`)
	output := reHeaders.ReplaceAllStringFunc(input, func(m string) string {
		title := reHeaders.FindStringSubmatch(m)[1]
		return "\n**" + strings.ToUpper(strings.TrimSpace(title)) + "**\n"
	})

	// 2. Identify potential headers ending in ':' and bold them if not already
	// E.g. "Summary:" -> "**Summary:**"
	reColons := regexp.MustCompile(`(?m)^([^#\n\*]{2,60}:)$`)
	output = reColons.ReplaceAllString(output, "\n**$1**\n")

	// 3. Ensure a blank line before bullet points if they follow text
	reLists := regexp.MustCompile(`([^\n])\n- `)
	output = reLists.ReplaceAllString(output, "$1\n\n- ")

	// 4. Whitespace Normalization
	lines := strings.Split(output, "\n")
	var finalLines []string
	for _, line := range lines {
		finalLines = append(finalLines, strings.TrimSpace(line))
	}
	output = strings.Join(finalLines, "\n")

	// Collapse 3+ newlines into exactly 2
	reExcessive := regexp.MustCompile(`\n{3,}`)
	output = reExcessive.ReplaceAllString(output, "\n\n")

	// 5. Final cleanup
	return strings.TrimSpace(output)
}
