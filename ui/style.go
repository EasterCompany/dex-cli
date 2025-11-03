package ui

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
)

const (
	ColorRed        = "\033[31m"
	ColorBrightRed  = "\033[91m"
	ColorGreen      = "\033[32m"
	ColorYellow     = "\033[33m"
	ColorBlue       = "\033[34m"
	ColorPurple     = "\033[35m"
	ColorCyan       = "\033[36m"
	ColorWhite      = "\033[37m"
	ColorDarkGray   = "\033[90m"
	ColorReset      = "\033[0m"
)

// ansiRegex is used to strip ANSI escape codes.
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// StripANSI removes ANSI color codes from a string.
func StripANSI(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}

// --- Basic Printing ---

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

// --- Update & Version Specific Helpers ---

type TagInfo struct {
	Latest string `json:"latest"`
}

type TagsMap map[string]TagInfo

var (
	cachedLatestVersion *string
	cacheInitialized    bool
)

func PrintSection(title string) {
	fmt.Printf("\n%s=== %s ===%s\n", ColorCyan, title, ColorReset)
}

func PrintVersionComparison(oldVersion, newVersion, latestVersion, buildYear string, oldSize, newSize int64, additions, deletions int) {
	fmt.Printf("%s  Previous version: %s%s\n", ColorBlue, ColorReset, FormatVersionWithTrademark(oldVersion, buildYear))
	fmt.Printf("%s  Current version:  %s%s\n", ColorBlue, ColorReset, FormatVersionWithTrademark(newVersion, buildYear))
	if latestVersion != "" {
		fmt.Printf("%s  Latest version:   %s%s\n", ColorBlue, ColorReset, FormatVersionWithTrademark(latestVersion, buildYear))
	}

	sizeDiff := newSize - oldSize
	var sizeColor, sizeIndicator string
	if sizeDiff > 0 {
		sizeColor, sizeIndicator = ColorRed, "↑"
	} else if sizeDiff < 0 {
		sizeColor, sizeIndicator = ColorGreen, "↓"
	} else {
		sizeColor, sizeIndicator = ColorYellow, "="
	}

	fmt.Printf("%s  Binary size:      %s%s → %s (%s%s %s%s)\n",
		ColorBlue, ColorReset, formatBytes(oldSize), formatBytes(newSize),
		sizeColor, sizeIndicator, formatBytes(abs(sizeDiff)), ColorReset)

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

func FetchLatestVersion() string {
	if cacheInitialized {
		if cachedLatestVersion != nil {
			return *cachedLatestVersion
		}
		return ""
	}
	cacheInitialized = true

	resp, err := http.Get("https://easter.company/tags/dex-cli.json")
	if err != nil {
		return ""
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}

	var tagInfo TagInfo
	if err := json.Unmarshal(body, &tagInfo); err == nil && tagInfo.Latest != "" {
		latest := strings.TrimSpace(tagInfo.Latest)
		cachedLatestVersion = &latest
		return latest
	}

	return ""
}

func IsOfficialRelease(fullVersion string) bool {
	latest := FetchLatestVersion()
	return latest != "" && latest == fullVersion
}

func FormatVersionWithTrademark(version string, buildYear string) string {
	if IsOfficialRelease(version) {
		return fmt.Sprintf("%s | Easter Company™ © %s", version, buildYear)
	}
	return version
}
