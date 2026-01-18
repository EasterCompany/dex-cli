package git

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// ... (existing code) ...

// ParseVersionTag parses a git tag string (e.g., "1.2.3") into its major, minor, and patch components.
func ParseVersionTag(tag string) (int, int, int, error) {
	trimmedTag := strings.TrimPrefix(tag, "v") // Still trim just in case, but don't expect it.
	parts := strings.Split(trimmedTag, ".")
	if len(parts) != 3 {
		return 0, 0, 0, fmt.Errorf("tag does not have 3 parts separated by '.'")
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to parse major version '%s': %w", parts[0], err)
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to parse minor version '%s': %w", parts[1], err)
	}
	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to parse patch version '%s': %w", parts[2], err)
	}

	return major, minor, patch, nil
}

// Version holds the components of a parsed version string.
type Version struct {
	Major     string
	Minor     string
	Patch     string
	Branch    string
	Commit    string
	BuildDate string
	Arch      string
	BuildHash string
}

// Parse takes a version string and returns a Version object.
func Parse(versionStr string) (*Version, error) {
	versionStr = strings.TrimPrefix(versionStr, "v") // Always trim 'v' if present
	parts := strings.Split(versionStr, ".")

	// Handle simple "M.m.p" versions, common for cache services or initial states.
	if len(parts) == 3 {
		return &Version{
			Major: parts[0],
			Minor: parts[1],
			Patch: parts[2],
		}, nil
	}

	// Handle the full "M.m.p.branch.commit.build_date.arch" format (7 parts)
	// or the extended version with build_hash (8 parts)
	if len(parts) != 7 && len(parts) != 8 {
		return nil, fmt.Errorf("invalid version string format: expected 3, 7, or 8 parts, got %d for '%s'", len(parts), versionStr)
	}

	v := &Version{
		Major:     parts[0],
		Minor:     parts[1],
		Patch:     parts[2],
		Branch:    parts[3],
		Commit:    parts[4],
		BuildDate: parts[5],
		Arch:      parts[6],
	}

	if len(parts) == 8 {
		v.BuildHash = parts[7]
	}

	return v, nil
}

// String formats a Version object back into a version string.
func (v *Version) String() string {
	// If we only have major/minor/patch, return the short form.
	if v.Branch == "" || v.Commit == "" {
		return v.Short()
	}
	if v.BuildHash == "" {
		return fmt.Sprintf("%s.%s.%s.%s.%s.%s.%s",
			v.Major, v.Minor, v.Patch,
			v.Branch, v.Commit, v.BuildDate, v.Arch,
		)
	}
	return fmt.Sprintf("%s.%s.%s.%s.%s.%s.%s.%s",
		v.Major, v.Minor, v.Patch,
		v.Branch, v.Commit, v.BuildDate, v.Arch, v.BuildHash,
	)
}

// Short returns the MAJOR.MINOR.PATCH part of the version.
func (v *Version) Short() string {
	return fmt.Sprintf("%s.%s.%s", v.Major, v.Minor, v.Patch)
}

// Compare compares two Version objects based on their MAJOR.MINOR.PATCH numbers.
// It returns:
// -1 if v is less than other
//
//	0 if v is equal to other
//	1 if v is greater than other
func (v *Version) Compare(other *Version) int {
	vMajor, _ := strconv.Atoi(v.Major)
	oMajor, _ := strconv.Atoi(other.Major)
	if vMajor != oMajor {
		if vMajor < oMajor {
			return -1
		}
		return 1
	}

	vMinor, _ := strconv.Atoi(v.Minor)
	oMinor, _ := strconv.Atoi(other.Minor)
	if vMinor != oMinor {
		if vMinor < oMinor {
			return -1
		}
		return 1
	}

	vPatch, _ := strconv.Atoi(v.Patch)
	oPatch, _ := strconv.Atoi(other.Patch)
	if vPatch != oPatch {
		if vPatch < oPatch {
			return -1
		}
		return 1
	}

	return 0
}

// FetchLatestVersionFromURL fetches the latest version string from a URL.
// Returns the full version string (e.g., "0.0.1.main.84a41d9...") or an error.
func FetchLatestVersionFromURL(url string) (string, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch version from %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code %d from %s", resp.StatusCode, url)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	version := strings.TrimSpace(string(body))
	if version == "" {
		return "", fmt.Errorf("empty version string from %s", url)
	}

	return version, nil
}

// IncrementVersion takes a version and increment type, returns the new incremented version.
// incrementType can be "major", "minor", or "patch"
// Returns (newMajor, newMinor, newPatch)
func IncrementVersion(major, minor, patch int, incrementType string) (int, int, int, error) {
	switch incrementType {
	case "major":
		return major + 1, 0, 0, nil
	case "minor":
		return major, minor + 1, 0, nil
	case "patch":
		return major, minor, patch + 1, nil
	default:
		return 0, 0, 0, fmt.Errorf("invalid increment type: %s (must be 'major', 'minor', or 'patch')", incrementType)
	}
}
