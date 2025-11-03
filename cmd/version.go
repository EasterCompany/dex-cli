package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// TagInfo represents the structure of a single tag entry in the JSON
type TagInfo struct {
	Latest string `json:"latest"`
}

// TagsMap represents the overall JSON structure
type TagsMap map[string]TagInfo

func Version(version, branch, commit, buildDate, buildYear string) {
	// Style for the trademark part
	darkStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240")) // Dark grey

	// Format the build date: 2025-11-03T02:38:13Z -> 2025-11-03-02-38-13
	formattedDate := strings.ReplaceAll(buildDate, "T", "-")
	formattedDate = strings.ReplaceAll(formattedDate, ":", "-")
	formattedDate = strings.TrimSuffix(formattedDate, "Z")

	// Format the architecture: linux/amd64 -> linux_amd64
	arch := fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH)

	// Create the full version string
	fullVersion := fmt.Sprintf("v%s.%s.%s.%s.%s",
		strings.TrimPrefix(version, "v"),
		branch,
		commit,
		formattedDate,
		arch,
	)

	// Conditionally add the trademark part
	trademarkPart := ""
	if isOfficialRelease(fullVersion) {
		trademarkPart = darkStyle.Render(fmt.Sprintf("| Easter Company™ © %s", buildYear))
	}

	// Assemble and print the final output
	if trademarkPart != "" {
		fmt.Printf("%s %s\n", fullVersion, trademarkPart)
	} else {
		fmt.Printf("%s\n", fullVersion)
	}
}

func isOfficialRelease(fullVersion string) bool {
	resp, err := http.Get("https://easter.company/tags/dex-cli.json")
	if err != nil {
		return false
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return false
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false
	}

	var tagsMap TagsMap
	if err := json.Unmarshal(body, &tagsMap); err != nil {
		return false
	}

	if tagInfo, ok := tagsMap["dex-cli"]; ok {
		if strings.TrimSpace(tagInfo.Latest) == fullVersion {
			return true
		}
	}

	return false
}
