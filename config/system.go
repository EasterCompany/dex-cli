package config

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// SystemConfig represents the hardware and software state of the system
type SystemConfig struct {
	MemoryBytes int64     `json:"MEMORY_BYTES"`
	CPU         []CPUInfo `json:"CPU"`
	GPU         []GPUInfo `json:"GPU"`
	Storage     []string  `json:"STORAGE"`
	Packages    []Package `json:"PACKAGES"`
	LastUpdated int64     `json:"LAST_UPDATED,omitempty"`
}

type CPUInfo struct {
	Label   string  `json:"LABEL"`
	Count   int     `json:"COUNT"`
	Threads int     `json:"THREADS"`
	AvgGHz  float64 `json:"AVG_GHZ"`
	MaxGHz  float64 `json:"MAX_GHZ"`
}

type GPUInfo struct {
	Label            string `json:"LABEL"`
	CUDA             int    `json:"CUDA"`
	VRAM             int64  `json:"VRAM"`
	ComputePriority  int    `json:"COMPUTE_PRIORITY"`
	ComputePotential int    `json:"COMPUTE_POTENTIAL"`
}

type Package struct {
	Name           string `json:"name"`
	Version        string `json:"version"`
	Required       bool   `json:"required"`
	MinVersion     string `json:"min_version,omitempty"`
	Installed      bool   `json:"installed"`
	InstallCommand string `json:"install_command,omitempty"`
}

// RequiredPackages defines packages needed for Dexter
var RequiredPackages = []Package{
	{Name: "git", Required: true, MinVersion: "2.0", InstallCommand: "sudo pacman -S git || sudo apt install git"},
	{Name: "go", Required: true, MinVersion: "1.20", InstallCommand: "sudo pacman -S go || sudo apt install golang-go"},
	{Name: "python3", Required: true, MinVersion: "3.10", InstallCommand: "sudo pacman -S python || sudo apt install python3"},
	{Name: "redis-server", Required: false, MinVersion: "6.0", InstallCommand: "sudo pacman -S redis || sudo apt install redis-server"},
}

// LoadSystemConfig loads or creates system.json
func LoadSystemConfig() (*SystemConfig, error) {
	configPath, err := ExpandPath(filepath.Join(DexterRoot, "config", "system.json"))
	if err != nil {
		return nil, err
	}

	// Try to load existing
	if data, err := os.ReadFile(configPath); err == nil {
		var sys SystemConfig
		if err := json.Unmarshal(data, &sys); err == nil {
			return &sys, nil
		}
	}

	// Create new
	return IntrospectSystem()
}

// IntrospectSystem scans hardware and software
func IntrospectSystem() (*SystemConfig, error) {
	sys := &SystemConfig{
		CPU:      detectCPU(),
		GPU:      detectGPU(),
		Storage:  detectStorage(),
		Packages: detectPackages(),
	}

	// Detect RAM
	sys.MemoryBytes = detectRAM()

	return sys, nil
}

// SaveSystemConfig saves system.json
func SaveSystemConfig(sys *SystemConfig) error {
	configPath, err := ExpandPath(filepath.Join(DexterRoot, "config", "system.json"))
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(sys, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

// detectCPU introspects CPU information
func detectCPU() []CPUInfo {
	threads := runtime.NumCPU()
	cores := threads / 2 // Assume hyperthreading

	cpuInfo := CPUInfo{
		Label:   "Unknown CPU",
		Count:   cores,
		Threads: threads,
		AvgGHz:  0,
		MaxGHz:  0,
	}

	// Try to get CPU model and core count from /proc/cpuinfo on Linux
	if runtime.GOOS == "linux" {
		if data, err := os.ReadFile("/proc/cpuinfo"); err == nil {
			lines := strings.Split(string(data), "\n")
			coreIDs := make(map[string]bool)

			for _, line := range lines {
				if strings.HasPrefix(line, "model name") {
					parts := strings.Split(line, ":")
					if len(parts) > 1 {
						cpuInfo.Label = strings.TrimSpace(parts[1])
					}
				}
				// Count unique physical cores
				if strings.HasPrefix(line, "core id") {
					parts := strings.Split(line, ":")
					if len(parts) > 1 {
						coreIDs[strings.TrimSpace(parts[1])] = true
					}
				}
			}

			// If we found core IDs, use that count
			if len(coreIDs) > 0 {
				cpuInfo.Count = len(coreIDs)
			}
		}
	}

	return []CPUInfo{cpuInfo}
}

// detectGPU introspects GPU information
func detectGPU() []GPUInfo {
	var gpus []GPUInfo

	// Try nvidia-smi for NVIDIA GPUs
	cmd := exec.Command("nvidia-smi", "--query-gpu=name,memory.total", "--format=csv,noheader")
	if output, err := cmd.Output(); err == nil {
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		for i, line := range lines {
			parts := strings.Split(line, ",")
			if len(parts) >= 2 {
				name := strings.TrimSpace(parts[0])
				vramStr := strings.TrimSpace(strings.Split(parts[1], " ")[0])
				vram, _ := strconv.ParseInt(vramStr, 10, 64)
				vram *= 1024 * 1024 // Convert MB to bytes

				gpus = append(gpus, GPUInfo{
					Label:            name,
					CUDA:             1, // Assume CUDA available if nvidia-smi works
					VRAM:             vram,
					ComputePriority:  i,
					ComputePotential: 1,
				})
			}
		}
	}

	// If no GPUs detected, return empty
	return gpus
}

// detectRAM gets total system RAM
func detectRAM() int64 {
	if runtime.GOOS == "linux" {
		if data, err := os.ReadFile("/proc/meminfo"); err == nil {
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "MemTotal:") {
					fields := strings.Fields(line)
					if len(fields) >= 2 {
						kb, _ := strconv.ParseInt(fields[1], 10, 64)
						return kb * 1024 // Convert KB to bytes
					}
				}
			}
		}
	}
	return 0
}

// detectStorage lists storage devices
func detectStorage() []string {
	var storage []string

	// Try lsblk on Linux
	if runtime.GOOS == "linux" {
		cmd := exec.Command("lsblk", "-d", "-n", "-o", "NAME,SIZE,TYPE")
		if output, err := cmd.Output(); err == nil {
			lines := strings.Split(strings.TrimSpace(string(output)), "\n")
			for _, line := range lines {
				if strings.Contains(line, "disk") {
					storage = append(storage, strings.TrimSpace(line))
				}
			}
		}
	}

	return storage
}

// detectPackages checks which required packages are installed
func detectPackages() []Package {
	packages := make([]Package, len(RequiredPackages))
	copy(packages, RequiredPackages)

	for i := range packages {
		pkg := &packages[i]

		// Check if package exists
		_, err := exec.LookPath(pkg.Name)
		pkg.Installed = err == nil

		// Try to get version
		if pkg.Installed {
			pkg.Version = getPackageVersion(pkg.Name)
		}
	}

	return packages
}

// getPackageVersion tries to get package version
func getPackageVersion(pkgName string) string {
	versionArgs := map[string][]string{
		"git":          {"--version"},
		"go":           {"version"},
		"python3":      {"--version"},
		"redis-server": {"--version"},
	}

	args, ok := versionArgs[pkgName]
	if !ok {
		return "unknown"
	}

	cmd := exec.Command(pkgName, args...)
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}

	// Parse version from output
	versionStr := strings.TrimSpace(string(output))
	return versionStr
}

// ValidateAndHealConfig compares user config with expected schema
func ValidateAndHealConfig(userConfig, schemaConfig interface{}) (warnings []string, healed bool) {
	// This is where the magic happens - deep config validation
	// For now, return empty - we'll implement the deep diff logic
	return warnings, false
}

// DiffConfigs finds differences between two configs
func DiffConfigs(current, new interface{}) (added, removed, changed []string) {
	// Deep comparison logic
	// Returns fields that are:
	// - added: in new but not in current
	// - removed: in current but not in new (dead fields)
	// - changed: in both but values differ
	return added, removed, changed
}

// MergeConfigs intelligently merges two configs
func MergeConfigs(base, updates interface{}) (interface{}, error) {
	// Smart merge that preserves user values but adds new defaults
	return base, nil
}

// PrintSystemInfo displays system config in a nice format
func PrintSystemInfo(sys *SystemConfig) {
	fmt.Println("=== System Information ===")
	fmt.Println()

	// RAM
	ramGB := float64(sys.MemoryBytes) / (1024 * 1024 * 1024)
	fmt.Printf("RAM: %.1f GB\n", ramGB)
	fmt.Println()

	// CPU
	fmt.Println("CPU:")
	for _, cpu := range sys.CPU {
		fmt.Printf("  %s\n", cpu.Label)
		fmt.Printf("    Cores: %d | Threads: %d\n", cpu.Count, cpu.Threads)
	}
	fmt.Println()

	// GPU
	if len(sys.GPU) > 0 {
		fmt.Println("GPU:")
		for i, gpu := range sys.GPU {
			fmt.Printf("  [%d] %s\n", i, gpu.Label)
			if gpu.VRAM > 0 {
				vramGB := float64(gpu.VRAM) / (1024 * 1024 * 1024)
				fmt.Printf("      VRAM: %.1f GB | CUDA: %d\n", vramGB, gpu.CUDA)
			}
		}
		fmt.Println()
	}

	// Storage
	if len(sys.Storage) > 0 {
		fmt.Println("Storage:")
		for _, disk := range sys.Storage {
			fmt.Printf("  %s\n", disk)
		}
		fmt.Println()
	}

	// Packages
	fmt.Println("Packages:")
	for _, pkg := range sys.Packages {
		status := "✗"
		if pkg.Installed {
			status = "✓"
		}
		required := ""
		if pkg.Required {
			required = " (required)"
		}
		fmt.Printf("  %s %s: %s%s\n", status, pkg.Name, pkg.Version, required)
	}
	fmt.Println()
}
