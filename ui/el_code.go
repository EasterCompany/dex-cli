package ui

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Global regex patterns for simpler highlighting
var (
// Matches quoted strings in various styles (e.g., "str", 'str', `str`)
// stringPattern = regexp.MustCompile(`"(.*?)"|'(.*?)'|\x60(.*?)\x60`)
// Matches single-line comments that start with # or //, optionally preceded by whitespace.
// commentPattern = regexp.MustCompile(`(^[\s]*#.*)|(^[\s]*//.*)`)
)

// CodeSnippet represents the data needed to render the code viewer element.
type CodeSnippet struct {
	FileName    string
	SizeKB      float64
	Status      string // e.g., "no issues", "2 warnings", "1 error"
	CodeContent string
	Language    string // Used for basic syntax highlighting
}

// PrintCodeBlock renders a stylized, boxed code snippet with colorized syntax.
func PrintCodeBlock(snippet CodeSnippet) {
	// 1. Prepare Header Metadata (Top Border)
	// statusColor := ""
	// if strings.Contains(strings.ToLower(snippet.Status), "error") {
	// 	statusColor = ""
	// } else if strings.Contains(strings.ToLower(snippet.Status), "warning") {
	// 	statusColor = ""
	// }

	// Dynamic size display
	sizeStr := ""
	if snippet.SizeKB > 0 {
		sizeStr = fmt.Sprintf("%.1fkb", snippet.SizeKB)
	}

	metaLine := fmt.Sprintf("[ %s ] %s, %s",
		snippet.FileName,
		sizeStr,
		snippet.Status)

	// Horizontal separator line (Fixed width)
	separator := strings.Repeat(BorderHorizontal, 80)

	PrintRaw(fmt.Sprintf("%s\n", metaLine))
	PrintRaw(fmt.Sprintf("%s\n", separator))

	// 2. Process and Render Code Content
	lines := strings.Split(snippet.CodeContent, "\n")

	// Determine the required width for line numbers
	lineNumberWidth := len(fmt.Sprintf("%d", len(lines)))
	if lineNumberWidth < 2 {
		lineNumberWidth = 2
	}

	for i, line := range lines {
		// Line Number
		lineNumber := fmt.Sprintf("%*d", lineNumberWidth, i+1)

		// Code (Syntax Highlighted)
		highlightedCode := highlightSyntax(line, snippet.Language)

		// Output: Line Num + Space + Code
		PrintRaw(fmt.Sprintf("%s %s\n", lineNumber, highlightedCode))
	}

	// 3. Render Bottom Border
	PrintRaw(fmt.Sprintf("%s\n", separator))
}

// PrintCodeBlockFromBytes is a helper to print raw bytes (useful for JSON/YAML/etc.)
// It automatically pretty-prints JSON before sending it to the block renderer.
func PrintCodeBlockFromBytes(data []byte, filename string, language string) {
	content := string(data)

	// Pretty-print JSON if needed
	if strings.ToLower(language) == "json" {
		var v interface{}
		// Unmarshal and MarshalIndent to ensure clean, indented content
		if err := json.Unmarshal(data, &v); err == nil {
			if b, err := json.MarshalIndent(v, "", "  "); err == nil {
				content = string(b)
			}
		}
	}

	// Simple size calculation (approximation)
	sizeKB := float64(len(data)) / 1024.0

	snippet := CodeSnippet{
		FileName:    filename,
		SizeKB:      sizeKB,
		Status:      "no issues",
		CodeContent: content,
		Language:    language,
	}

	PrintCodeBlock(snippet)
}

// highlightSyntax applies rudimentary syntax highlighting based on language/context.
func highlightSyntax(line, language string) string {
	return line
}
