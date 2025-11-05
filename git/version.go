package git

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ... (existing code) ...

// ParseVersionTag parses a git tag string (e.g., "v1.2.3") into its major, minor, and patch components.
func ParseVersionTag(tag string) (int, int, int, error) {
	if !strings.HasPrefix(tag, "v") {
		return 0, 0, 0, fmt.Errorf("tag does not start with 'v'")
	}
	trimmedTag := strings.TrimPrefix(tag, "v")
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
	Major      string
	Minor      string
	Patch      string
	PreRelease string
	Branch     string
	Commit     string
	BuildDate  string
	Arch       string
	Random     string
}

// Parse takes a version string and returns a Version object.
func Parse(versionStr string) (*Version, error) {
	// Regex to capture all parts of the version string
	re := regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)(-[a-zA-Z0-9]+)?\.([a-zA-Z0-9_-]+)\.([a-f0-9]+)\.([0-9-]+)\.([a-zA-Z0-9_]+)\.([a-zA-Z0-9]+)$`)
	matches := re.FindStringSubmatch(versionStr)

	if len(matches) != 10 {
		// A simpler regex for just MAJOR.MINOR.PATCH, often from cache services
		reSimple := regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)$`)
		matchesSimple := reSimple.FindStringSubmatch(versionStr)
		if len(matchesSimple) != 4 {
			return nil, fmt.Errorf("invalid version string format: %s", versionStr)
		}
		return &Version{
			Major: matchesSimple[1],
			Minor: matchesSimple[2],
			Patch: matchesSimple[3],
		}, nil
	}

	return &Version{
		Major:      matches[1],
		Minor:      matches[2],
		Patch:      matches[3],
		PreRelease: strings.TrimPrefix(matches[4], "-"),
		Branch:     matches[5],
		Commit:     matches[6],
		BuildDate:  matches[7],
		Arch:       matches[8],
		Random:     matches[9],
	}, nil
}

// String formats a Version object back into a version string.
func (v *Version) String() string {
	// If we only have major/minor/patch, return the short form
	if v.Branch == "" {
		return v.Short()
	}
	preRelease := ""
	if v.PreRelease != "" {
		preRelease = fmt.Sprintf("-%s", v.PreRelease)
	}
	return fmt.Sprintf("v%s.%s.%s%s.%s.%s.%s.%s.%s",
		v.Major, v.Minor, v.Patch, preRelease,
		v.Branch, v.Commit, v.BuildDate, v.Arch, v.Random,
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
