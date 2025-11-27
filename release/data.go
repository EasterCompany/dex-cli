package release

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// ReleaseData represents the complete /bin/data.json structure
type ReleaseData struct {
	Latest   LatestVersions         `json:"latest"`
	Releases map[string]ReleaseInfo `json:"releases"`
	Services map[string]ServiceInfo `json:"services"`
}

// LatestVersions tracks current user and dev versions
// Stores FULL version strings including build hash for exact matching
type LatestVersions struct {
	User string `json:"user"` // Latest user version (full: 2.1.0.main.abc123.2025-11-27-09-30-45.linux-amd64.xyz789)
	Dev  string `json:"dev"`  // Latest dev version (full: 2.1.3.main.def456.2025-11-27-10-15-20.linux-amd64.qrs234)
}

// ReleaseInfo contains metadata about a specific release
type ReleaseInfo struct {
	Type     string                       `json:"type"`     // "major" or "minor"
	Date     string                       `json:"date"`     // ISO 8601
	Commit   string                       `json:"commit"`   // Git commit hash
	Binaries map[string]map[string]Binary `json:"binaries"` // service -> platform -> binary
}

// Binary contains info about a specific binary file
type Binary struct {
	Path     string `json:"path"`     // Relative path from root
	Size     int64  `json:"size"`     // Size in bytes
	Checksum string `json:"checksum"` // SHA256
}

// ServiceInfo tracks per-service version information
type ServiceInfo struct {
	Current    string `json:"current"`    // Current dev version (full string)
	User       string `json:"user"`       // Latest user-facing version (full string)
	Repository string `json:"repository"` // GitHub repo URL
}

// LoadReleaseData loads existing data.json or creates a new one
func LoadReleaseData(path string) (*ReleaseData, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty structure
			return &ReleaseData{
				Latest:   LatestVersions{},
				Releases: make(map[string]ReleaseInfo),
				Services: make(map[string]ServiceInfo),
			}, nil
		}
		return nil, err
	}

	var rd ReleaseData
	if err := json.Unmarshal(data, &rd); err != nil {
		return nil, err
	}

	// Initialize maps if nil
	if rd.Releases == nil {
		rd.Releases = make(map[string]ReleaseInfo)
	}
	if rd.Services == nil {
		rd.Services = make(map[string]ServiceInfo)
	}

	return &rd, nil
}

// Save writes the data to disk
func (rd *ReleaseData) Save(path string) error {
	data, err := json.MarshalIndent(rd, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// AddRelease adds or updates a release in the data
func (rd *ReleaseData) AddRelease(version, releaseType, commit string) {
	rd.Releases[version] = ReleaseInfo{
		Type:     releaseType,
		Date:     time.Now().UTC().Format(time.RFC3339),
		Commit:   commit,
		Binaries: make(map[string]map[string]Binary),
	}
}

// UpdateService updates service version information
func (rd *ReleaseData) UpdateService(serviceName, currentVersion, userVersion, repo string) {
	rd.Services[serviceName] = ServiceInfo{
		Current:    currentVersion,
		User:       userVersion,
		Repository: repo,
	}
}

// AddBinary adds a binary to a release
func (rd *ReleaseData) AddBinary(version, service, platform, binaryPath string) error {
	release, exists := rd.Releases[version]
	if !exists {
		return fmt.Errorf("release %s not found", version)
	}

	// Calculate checksum
	checksum, err := CalculateChecksum(binaryPath)
	if err != nil {
		return fmt.Errorf("failed to calculate checksum: %w", err)
	}

	// Get file size
	info, err := os.Stat(binaryPath)
	if err != nil {
		return fmt.Errorf("failed to stat binary: %w", err)
	}

	// Get relative path from easter.company root
	relativePath := filepath.Join("/bin", version, filepath.Base(binaryPath))

	// Initialize maps if needed
	if release.Binaries[service] == nil {
		release.Binaries[service] = make(map[string]Binary)
	}

	release.Binaries[service][platform] = Binary{
		Path:     relativePath,
		Size:     info.Size(),
		Checksum: checksum,
	}

	rd.Releases[version] = release
	return nil
}

// RemoveMinorVersions removes all minor versions from the same major
func (rd *ReleaseData) RemoveMinorVersions(majorVersion string) []string {
	var removed []string
	for version, info := range rd.Releases {
		// Skip if it's a major version
		if info.Type == "major" {
			continue
		}
		// Check if it's from the same major version
		if len(version) >= len(majorVersion) && version[:len(majorVersion)] == majorVersion {
			delete(rd.Releases, version)
			removed = append(removed, version)
		}
	}
	return removed
}

// CalculateChecksum computes SHA256 of a file
func CalculateChecksum(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}
