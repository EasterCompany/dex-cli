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

	// 1. Remove Numbered Prefixes from lines that look like headers
	// E.g. "1. Political Landscape" -> "Political Landscape"
	reNumbering := regexp.MustCompile(`(?m)^\d+\.\s+`)
	output := reNumbering.ReplaceAllString(input, "")

	// 2. Convert '#' headers to **BOLD UPPERCASE** and ensure compact spacing
	// Spacing: \n\n**HEADER**\n (No blank line after header)
	reHeaders := regexp.MustCompile(`(?m)^#+\s+(.*)$`)
	output = reHeaders.ReplaceAllStringFunc(output, func(m string) string {
		title := reHeaders.FindStringSubmatch(m)[1]
		return "\n\n**" + strings.ToUpper(strings.TrimSpace(title)) + "**\n"
	})

	// 3. Identify potential headers ending in ':' and bold them if not already
	reColons := regexp.MustCompile(`(?m)^([^#\n\*]{2,60}:)$`)
	output = reColons.ReplaceAllString(output, "\n\n**$1**\n")

	// 4. Remove trailing commas from headers (LLM JSON artifact cleanup)
	reCommaHeader := regexp.MustCompile(`(?m)(\*\*.*:?)\s*,\s*$`)
	output = reCommaHeader.ReplaceAllString(output, "$1")

	// 5. Ensure a blank line before bullet points if they follow text
	reLists := regexp.MustCompile(`([^\n])\n- `)
	output = reLists.ReplaceAllString(output, "$1\n\n- ")

	// 6. Whitespace Normalization
	lines := strings.Split(output, "\n")
	var finalLines []string
	for _, line := range lines {
		finalLines = append(finalLines, strings.TrimSpace(line))
	}
	output = strings.Join(finalLines, "\n")

	// Collapse 3+ newlines into exactly 2
	reExcessive := regexp.MustCompile(`\n{3,}`)
	output = reExcessive.ReplaceAllString(output, "\n\n")

	return strings.TrimSpace(output)
}
