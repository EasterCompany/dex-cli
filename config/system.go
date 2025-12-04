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
	MemoryBytes int64         `json:"MEMORY_BYTES"`
	CPU         []CPUInfo     `json:"CPU"`
	GPU         []GPUInfo     `json:"GPU"`
	Storage     []StorageInfo `json:"STORAGE"`
	Packages    []Package     `json:"PACKAGES"`
	LastUpdated int64         `json:"LAST_UPDATED,omitempty"`
}

type StorageInfo struct {
	Device     string `json:"DEVICE"`
	Size       int64  `json:"SIZE"`
	Used       int64  `json:"USED"`
	MountPoint string `json:"MOUNT_POINT"`
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
	UpgradeCommand string `json:"upgrade_command,omitempty"`
}

// RequiredPackages defines packages needed for Dexter
var RequiredPackages = []Package{
	{Name: "git", Required: true, MinVersion: "2.0", InstallCommand: "sudo pacman -S --noconfirm git || sudo apt install -y git", UpgradeCommand: "sudo pacman -Syu --noconfirm git || (sudo apt update && sudo apt upgrade -y git)"},
	{Name: "go", Required: true, MinVersion: "1.20", InstallCommand: "sudo pacman -S --noconfirm go || sudo apt install -y golang-go", UpgradeCommand: "sudo pacman -Syu --noconfirm go || (sudo apt update && sudo apt upgrade -y golang-go)"},
	{Name: "python3", Required: true, MinVersion: "3.13", InstallCommand: "sudo pacman -S --noconfirm python || sudo apt install -y python3", UpgradeCommand: "sudo pacman -Syu --noconfirm python || (sudo apt update && sudo apt upgrade -y python3)"},
	{Name: "bun", Required: true, MinVersion: "1.0", InstallCommand: "curl -fsSL https://bun.sh/install | bash", UpgradeCommand: "bun upgrade"},
	{Name: "golangci-lint", Required: true, MinVersion: "1.50", InstallCommand: "go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest", UpgradeCommand: "go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"},
	{Name: "make", Required: true, MinVersion: "4.0", InstallCommand: "sudo pacman -S --noconfirm make || sudo apt install -y make", UpgradeCommand: "sudo pacman -Syu --noconfirm make || (sudo apt update && sudo apt upgrade -y make)"},
	{Name: "pip3", Required: true, MinVersion: "20.0", InstallCommand: "sudo pacman -S --noconfirm python-pip || sudo apt install -y python3-pip", UpgradeCommand: "pip3 install --upgrade pip"},
	{Name: "lsblk", Required: true, MinVersion: "", InstallCommand: "sudo pacman -S --noconfirm util-linux || sudo apt install -y util-linux", UpgradeCommand: "sudo pacman -Syu --noconfirm util-linux || (sudo apt update && sudo apt upgrade -y util-linux)"},
	{Name: "findmnt", Required: true, MinVersion: "", InstallCommand: "sudo pacman -S --noconfirm util-linux || sudo apt install -y util-linux", UpgradeCommand: "sudo pacman -Syu --noconfirm util-linux || (sudo apt update && sudo apt upgrade -y util-linux)"},
	{Name: "redis-server", Required: false, MinVersion: "6.0", InstallCommand: "sudo pacman -S --noconfirm redis || sudo apt install -y redis-server", UpgradeCommand: "sudo pacman -Syu --noconfirm redis || (sudo apt update && sudo apt upgrade -y redis-server)"},
	{Name: "htop", Required: false, MinVersion: "3.0", InstallCommand: "sudo pacman -S --noconfirm htop || sudo apt install -y htop", UpgradeCommand: "sudo pacman -Syu --noconfirm htop || (sudo apt update && sudo apt upgrade -y htop)"},
	{Name: "curl", Required: false, MinVersion: "7.0", InstallCommand: "sudo pacman -S --noconfirm curl || sudo apt install -y curl", UpgradeCommand: "sudo pacman -Syu --noconfirm curl || (sudo apt update && sudo apt upgrade -y curl)"},
	{Name: "jq", Required: false, MinVersion: "1.6", InstallCommand: "sudo pacman -S --noconfirm jq || sudo apt install -y jq", UpgradeCommand: "sudo pacman -Syu --noconfirm jq || (sudo apt update && sudo apt upgrade -y jq)"},
	{Name: "prettier", Required: true, InstallCommand: "bun install -g prettier", UpgradeCommand: "bun install -g prettier"},
	{Name: "eslint", Required: true, InstallCommand: "bun install -g eslint", UpgradeCommand: "bun install -g eslint"},
	{Name: "stylelint", Required: true, InstallCommand: "bun install -g stylelint", UpgradeCommand: "bun install -g stylelint"},
	{Name: "htmlhint", Required: true, InstallCommand: "bun install -g htmlhint", UpgradeCommand: "bun install -g htmlhint"},
	{Name: "jsonlint", Required: true, InstallCommand: "bun install -g jsonlint", UpgradeCommand: "bun install -g jsonlint"},
	{Name: "shfmt", Required: true, InstallCommand: "go install mvdan.cc/sh/v3/cmd/shfmt@latest", UpgradeCommand: "go install mvdan.cc/sh/v3/cmd/shfmt@latest"},
	{Name: "shellcheck", Required: true, InstallCommand: "sudo pacman -S shellcheck || sudo apt install shellcheck", UpgradeCommand: "sudo pacman -Syu shellcheck || (sudo apt update && sudo apt upgrade -y shellcheck)"},
	{Name: "nvidia-smi", Required: true, MinVersion: "", InstallCommand: "Install NVIDIA drivers via your distribution's package manager (e.g. sudo pacman -S nvidia-utils)", UpgradeCommand: "Update system packages"},
	{Name: "nvcc", Required: true, MinVersion: "11.0", InstallCommand: "sudo pacman -S cuda || sudo apt install nvidia-cuda-toolkit", UpgradeCommand: "sudo pacman -Syu cuda || (sudo apt update && sudo apt upgrade -y nvidia-cuda-toolkit)"},
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
		Packages: detectPackages(RequiredPackages),
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
			var frequencies []float64

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
				// Collect CPU frequencies
				if strings.HasPrefix(line, "cpu MHz") {
					parts := strings.Split(line, ":")
					if len(parts) > 1 {
						mhzStr := strings.TrimSpace(parts[1])
						if mhz, err := strconv.ParseFloat(mhzStr, 64); err == nil {
							frequencies = append(frequencies, mhz/1000.0) // Convert MHz to GHz
						}
					}
				}
			}

			// If we found core IDs, use that count
			if len(coreIDs) > 0 {
				cpuInfo.Count = len(coreIDs)
			}

			// Calculate average and max frequencies
			if len(frequencies) > 0 {
				var sum float64
				maxFreq := frequencies[0]
				for _, freq := range frequencies {
					sum += freq
					if freq > maxFreq {
						maxFreq = freq
					}
				}
				cpuInfo.AvgGHz = sum / float64(len(frequencies))
				cpuInfo.MaxGHz = maxFreq
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

// detectStorage lists storage devices with usage information
func detectStorage() []StorageInfo {
	var storage []StorageInfo

	// Try lsblk on Linux
	if runtime.GOOS != "linux" {
		return storage
	}

	// Get all block devices with their partitions
	cmd := exec.Command("lsblk", "-b", "-n", "-o", "NAME,SIZE,FSUSED,TYPE")
	output, err := cmd.Output()
	if err != nil {
		return storage
	}

	// Parse lsblk output to find disk devices
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	deviceMap := make(map[string]*StorageInfo) // Map device name to storage info
	partitionMap := make(map[string]string)    // Map partition to parent disk

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		name := strings.TrimLeft(fields[0], "├─└│")
		sizeStr := fields[1]
		devType := fields[len(fields)-1]

		// Parse size
		size, _ := strconv.ParseInt(sizeStr, 10, 64)

		// For disk devices (not partitions), initialize entry
		switch devType {
		case "disk":
			deviceMap[name] = &StorageInfo{
				Device:     name,
				Size:       size,
				Used:       0,
				MountPoint: "",
			}
		case "part":
			// Map partition to parent disk
			parentDisk := strings.TrimRight(name, "0123456789")
			partitionMap[name] = parentDisk
		}
	}

	// Use findmnt to get mount points and usage for each partition
	findmntCmd := exec.Command("findmnt", "-b", "-n", "-o", "TARGET,SOURCE,USED")
	findmntOutput, err := findmntCmd.Output()
	if err == nil {
		findmntLines := strings.Split(strings.TrimSpace(string(findmntOutput)), "\n")

		for _, line := range findmntLines {
			fields := strings.Fields(line)
			if len(fields) < 2 {
				continue
			}

			mountPoint := fields[0]
			source := fields[1]
			var used int64
			if len(fields) >= 3 {
				used, _ = strconv.ParseInt(fields[2], 10, 64)
			}

			// Extract device name from source (e.g., /dev/sda2[/@] -> sda2)
			deviceName := strings.TrimPrefix(source, "/dev/")
			if idx := strings.Index(deviceName, "["); idx != -1 {
				deviceName = deviceName[:idx]
			}

			// Find parent disk for this partition
			parentDisk, exists := partitionMap[deviceName]
			if !exists {
				continue
			}

			info, exists := deviceMap[parentDisk]
			if !exists {
				continue
			}

			// Prioritize root mount point, otherwise shortest path
			if mountPoint == "/" {
				info.MountPoint = mountPoint
				info.Used = used
			} else if info.MountPoint != "/" {
				if info.MountPoint == "" || len(mountPoint) < len(info.MountPoint) {
					info.MountPoint = mountPoint
					info.Used = used
				}
			}
		}
	}

	// Convert map to slice
	for _, info := range deviceMap {
		if info.MountPoint == "" {
			info.MountPoint = "unmounted"
		}
		storage = append(storage, *info)
	}

	return storage
}

// detectPackages checks which required packages are installed
func detectPackages(requiredPackages []Package) []Package {
	packages := make([]Package, len(requiredPackages))
	copy(packages, requiredPackages)

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

	// Add Python venv check
	venvPath, err := ExpandPath(filepath.Join(DexterRoot, "python"))
	if err == nil {
		venvExists := false
		if info, err := os.Stat(venvPath); err == nil && info.IsDir() {
			// Check for activate script to confirm it's a venv
			activatePath := filepath.Join(venvPath, "bin", "activate")
			if _, err := os.Stat(activatePath); err == nil {
				venvExists = true
			}
		}

		packages = append(packages, Package{
			Name:           "dexter-venv",
			Version:        venvPath,
			Required:       true,
			Installed:      venvExists,
			InstallCommand: "python3 -m venv ~/Dexter/python",
		})
	}

	return packages
}

// getPackageVersion tries to get package version
func getPackageVersion(pkgName string) string {
	versionArgs := map[string][]string{
		"git":           {"--version"},
		"go":            {"version"},
		"python3":       {"--version"},
		"bun":           {"--version"},
		"golangci-lint": {"--version"},
		"make":          {"--version"},
		"pip3":          {"--version"},
		"redis-server":  {"--version"},
		"htop":          {"--version"},
		"curl":          {"--version"},
		"jq":            {"--version"},
		"nvidia-smi":    {"--version"},
		"nvcc":          {"--version"},
		"lsblk":         {"--version"},
		"findmnt":       {"--version"},
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
			sizeGB := float64(disk.Size) / (1024 * 1024 * 1024)
			usedGB := float64(disk.Used) / (1024 * 1024 * 1024)
			fmt.Printf("  %s: %.1fGB / %.1fGB (%s)\n", disk.Device, usedGB, sizeGB, disk.MountPoint)
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
