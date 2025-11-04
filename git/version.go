package git

import (
	"fmt"
	"regexp"
	"strings"
)

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
		return nil, fmt.Errorf("invalid version string format: %s", versionStr)
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
