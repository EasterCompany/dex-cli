package ui

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// Global regex patterns for simpler highlighting
var (
	// Matches quoted strings in various styles (e.g., "str", 'str', `str`)
	stringPattern = regexp.MustCompile(`"(.*?)"|'(.*?)'|\x60(.*?)\x60`)
	// Matches single-line comments that start with # or //, optionally preceded by whitespace.
	commentPattern = regexp.MustCompile(`(^[\s]*#.*)|(^[\s]*//.*)`)
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
	statusColor := ColorGreen
	if strings.Contains(strings.ToLower(snippet.Status), "error") {
		statusColor = ColorRed
	} else if strings.Contains(strings.ToLower(snippet.Status), "warning") {
		statusColor = ColorYellow
	}

	// Dynamic size display
	sizeStr := ""
	if snippet.SizeKB > 0 {
		sizeStr = fmt.Sprintf("%.1fkb", snippet.SizeKB)
	}

	metaLine := fmt.Sprintf("%s[ %s ] %s%s%s, %s%s%s",
		ColorCyan, snippet.FileName,
		ColorDarkGray, sizeStr, ColorReset,
		statusColor, snippet.Status, ColorReset)

	// Horizontal separator line (Fixed width)
	separator := strings.Repeat(BorderHorizontal, 80)

	PrintRaw(fmt.Sprintf("%s%s%s\n", ColorDarkGray, metaLine, ColorReset))
	PrintRaw(fmt.Sprintf("%s%s%s\n", ColorDarkGray, separator, ColorReset))

	// 2. Process and Render Code Content
	lines := strings.Split(snippet.CodeContent, "\n")

	// Determine the required width for line numbers
	lineNumberWidth := len(fmt.Sprintf("%d", len(lines)))
	if lineNumberWidth < 2 {
		lineNumberWidth = 2
	}

	for i, line := range lines {
		// Line Number (Purple)
		lineNumber := fmt.Sprintf("%s%*d%s", ColorPurple, lineNumberWidth, i+1, ColorReset)

		// Code (Syntax Highlighted)
		highlightedCode := highlightSyntax(line, snippet.Language)

		// Output: Line Num + Space + Code + Reset
		PrintRaw(fmt.Sprintf("%s %s %s\n", lineNumber, highlightedCode, ColorReset))
	}

	// 3. Render Bottom Border
	PrintRaw(fmt.Sprintf("%s%s%s\n", ColorDarkGray, separator, ColorReset))
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
	highlighted := line
	lang := strings.ToLower(language)

	// --- General/Common Highlights (Applied first for maximum coverage) ---

	// Comments (Go, JS, TS, Bash, Python) - Dark Gray
	if commentPattern.MatchString(line) {
		parts := strings.SplitN(line, "//", 2)
		if len(parts) == 2 {
			// Handle // comments
			highlighted = parts[0] + Colorize(fmt.Sprintf("//%s", parts[1]), ColorDarkGray)
		} else {
			// Handle # comments
			parts = strings.SplitN(line, "#", 2)
			// Exclude CSS/HTML IDs, but NOT Markdown headers
			if len(parts) == 2 && lang != "css" && lang != "html" && lang != "markdown" && lang != "md" {
				highlighted = parts[0] + Colorize(fmt.Sprintf("#%s", parts[1]), ColorDarkGray)
			}
		}
	}

	// --- Language Specific Highlights ---

	switch lang {
	case "json":
		// Placeholder-based replacement to prevent regex interference.
		placeholders := make(map[string]string)
		placeholderID := 0

		// 1. Replace all strings with placeholders
		highlighted = stringPattern.ReplaceAllStringFunc(highlighted, func(s string) string {
			placeholder := fmt.Sprintf("__STRING_%d__", placeholderID)
			placeholders[placeholder] = Colorize(s, ColorGreen) // Default to green (value)
			placeholderID++
			return placeholder
		})

		// 2. Identify which placeholders are keys and color them blue
		keyPattern := regexp.MustCompile(`(__STRING_\d+__):`)
		highlighted = keyPattern.ReplaceAllStringFunc(highlighted, func(s string) string {
			placeholder := strings.TrimRight(s, ":")
			// Overwrite the map entry with the blue key color
			placeholders[placeholder] = Colorize(StripANSI(placeholders[placeholder]), ColorBlue)
			return s // Return the placeholder with the colon
		})

		// 3. Highlight primitives (numbers, bools, null)
		primitivesPattern := regexp.MustCompile(`\b(-?\d+(\.\d+)?|true|false|null)\b`)
		highlighted = primitivesPattern.ReplaceAllStringFunc(highlighted, func(s string) string {
			if strings.Contains(s, ColorReset) {
				return s
			}
			color := ColorPurple // Default for numbers
			if s == "true" || s == "false" || s == "null" {
				color = ColorYellow
			}
			return Colorize(s, color)
		})

		// 4. Highlight structural elements
		structPattern := regexp.MustCompile(`[{}[\],:]`)
		highlighted = structPattern.ReplaceAllStringFunc(highlighted, func(s string) string {
			if strings.Contains(s, ColorReset) {
				return s
			}
			return Colorize(s, ColorCyan)
		})

		// 5. Restore all placeholders
		for p, v := range placeholders {
			highlighted = strings.ReplaceAll(highlighted, p, v)
		}

	case "go", "js", "ts", "python", "bash", "sh":
		// Keywords - Yellow
		keywords := map[string][]string{
			"go":     {"package", "import", "func", "var", "const", "if", "for", "range", "return", "switch", "case", "default", "go", "defer", "struct", "interface", "chan", "map", "new", "make"},
			"python": {"def", "class", "import", "from", "if", "elif", "else", "for", "in", "while", "return", "yield", "with", "as", "try", "except", "finally", "pass", "None", "True", "False"},
			"js":     {"function", "class", "import", "export", "const", "let", "var", "if", "else", "for", "in", "of", "while", "return", "switch", "case", "default", "new", "this", "async", "await"},
			"ts":     {"interface", "type", "public", "private", "protected", "readonly", "enum", "extends", "implements"}, // Additional TS keywords
			"bash":   {"if", "then", "else", "fi", "for", "in", "do", "done", "while", "exit", "function", "echo", "read", "export", "local"},
			"sh":     {"if", "then", "else", "fi", "for", "in", "do", "done", "while", "exit", "function", "echo", "read", "export", "local"},
		}

		// Types/Builtins/Functions - Cyan
		types := map[string][]string{
			"go":     {"string", "int", "bool", "error", "byte", "rune", "fmt", "os"},
			"python": {"str", "int", "float", "list", "dict", "tuple", "set", "print", "range", "len", "self", "super"},
			"js":     {"String", "Number", "Boolean", "Object", "Array", "console", "document", "window", "Promise"},
			"ts":     {"string", "number", "boolean", "any", "void", "never"},
			"bash":   {"$", "@", "*", "!"}, // Variables and special characters
			"sh":     {"$", "@", "*", "!"},
		}

		// Apply keywords
		if kws, ok := keywords[lang]; ok {
			for _, kw := range kws {
				// Use word boundary regex for precise replacement
				re := regexp.MustCompile(fmt.Sprintf(`\b%s\b`, regexp.QuoteMeta(kw)))
				highlighted = re.ReplaceAllStringFunc(highlighted, func(s string) string {
					// Only color if the text isn't already colored by comments or strings
					if strings.Contains(s, ColorReset) {
						return s
					}
					return Colorize(s, ColorYellow)
				})
			}
		}

		// Apply types/builtins/functions
		if tps, ok := types[lang]; ok {
			for _, tp := range tps {
				re := regexp.MustCompile(fmt.Sprintf(`\b%s\b`, regexp.QuoteMeta(tp)))
				highlighted = re.ReplaceAllStringFunc(highlighted, func(s string) string {
					if strings.Contains(s, ColorReset) {
						return s
					}
					return Colorize(s, ColorCyan)
				})
			}
		}

	case "html":
		// Tags (<tag>, </tag>) - Bright Red
		tagPattern := regexp.MustCompile(`(</?[\w-]+)`)
		highlighted = tagPattern.ReplaceAllStringFunc(highlighted, func(s string) string {
			return Colorize(s, ColorBrightRed)
		})

		// Attributes (attr=) - Cyan
		attrPattern := regexp.MustCompile(`([\w-]+)=`)
		highlighted = attrPattern.ReplaceAllStringFunc(highlighted, func(s string) string {
			return Colorize(strings.TrimRight(s, "="), ColorCyan) + "="
		})

	case "css":
		// Selectors (.class, #id, tag) - Yellow
		selectorPattern := regexp.MustCompile(`([\w\.\#:\*\s]+){`)
		highlighted = selectorPattern.ReplaceAllStringFunc(highlighted, func(s string) string {
			selector := strings.TrimRight(s, "{")
			return Colorize(selector, ColorYellow) + "{"
		})

		// Properties (property: ) - Cyan
		propPattern := regexp.MustCompile(`([\w-]+):`)
		highlighted = propPattern.ReplaceAllStringFunc(highlighted, func(s string) string {
			return Colorize(strings.TrimRight(s, ":"), ColorCyan) + ":"
		})

		// Values (simple values) - Purple
		// This is a simplified regex to target values after a colon, up to the semi-colon or end of line
		valuePattern := regexp.MustCompile(`:(\s*[^;]+)`)
		highlighted = valuePattern.ReplaceAllStringFunc(highlighted, func(s string) string {
			// Only color the part *after* the colon.
			parts := strings.SplitN(s, ":", 2)
			if len(parts) == 2 {
				return ":" + Colorize(parts[1], ColorPurple)
			}
			return s
		})

	case "markdown", "md":
		// Headers (#, ##, etc) - Bright Red for the markers
		headerPattern := regexp.MustCompile(`(^\s*#+\s)`)
		highlighted = headerPattern.ReplaceAllStringFunc(highlighted, func(s string) string {
			return Colorize(s, ColorBrightRed)
		})

		// Emphasis (Bold/Italic markers: **, __, *, _, and inline code backticks `) - Yellow
		emphasisPattern := regexp.MustCompile(`(\*\*|\*|__|_|\x60\x60\x60|\x60)`)
		highlighted = emphasisPattern.ReplaceAllStringFunc(highlighted, func(s string) string {
			// Only color the markers themselves
			return Colorize(s, ColorYellow)
		})

		// Links/Images ([text](url)) - Cyan for structure
		// Targets the entire link structure's brackets and parentheses
		linkBracketsPattern := regexp.MustCompile(`[\[\]]`) // The [ and ] part
		highlighted = linkBracketsPattern.ReplaceAllStringFunc(highlighted, func(s string) string {
			return Colorize(s, ColorCyan)
		})

		linkParenthesesPattern := regexp.MustCompile(`[\(\)]`) // The ( and ) part
		highlighted = linkParenthesesPattern.ReplaceAllStringFunc(highlighted, func(s string) string {
			return Colorize(s, ColorCyan)
		})

		// List markers (*, -, + and 1., 2. etc.) - Purple
		listPattern := regexp.MustCompile(`(^\s*[\*\-\+]\s)|(^\s*\d+\.\s)`)
		highlighted = listPattern.ReplaceAllStringFunc(highlighted, func(s string) string {
			return Colorize(s, ColorPurple)
		})

		// Blockquotes (>) - Dark Gray
		blockquotePattern := regexp.MustCompile(`(^\s*>+\s)`)
		highlighted = blockquotePattern.ReplaceAllStringFunc(highlighted, func(s string) string {
			return Colorize(s, ColorDarkGray)
		})

	case "text", "plain", "":
		// No highlighting for plain text or unknown
		return line

	default:
		// Default to plain text if language is not supported
		return line
	}

	// Apply general string highlighting for languages that don't have special handling
	if lang != "json" {
		highlighted = stringPattern.ReplaceAllStringFunc(highlighted, func(s string) string {
			// Do not re-color if already part of a comment (contains DarkGray)
			if strings.Contains(s, ColorDarkGray) {
				return s
			}
			return Colorize(s, ColorGreen)
		})
	}

	return highlighted
}

// HighlightAndColor takes a string of content and a language, and returns the
// fully highlighted and color-coded string.
func HighlightAndColor(content, language string) string {
	var sb strings.Builder
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		sb.WriteString(highlightSyntax(line, language) + "\n")
	}
	return sb.String()
}
