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
	{Name: "pip3", Required: true, MinVersion: "20.0", InstallCommand: "sudo pacman -S --noconfirm python-pip || sudo apt install -y python3-pip", UpgradeCommand: "~/Dexter/python/bin/python -m pip install --upgrade pip"},
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
	{Name: "nvidia-smi", Required: true, MinVersion: "", InstallCommand: "sudo pacman -S --noconfirm nvidia-utils || sudo apt install -y nvidia-utils", UpgradeCommand: "sudo pacman -Syu --noconfirm nvidia-utils || (sudo apt update && sudo apt upgrade -y nvidia-utils)"},
	{Name: "nvcc", Required: true, MinVersion: "11.0", InstallCommand: "sudo pacman -S cuda || sudo apt install nvidia-cuda-toolkit", UpgradeCommand: "sudo pacman -Syu cuda || (sudo apt update && sudo apt upgrade -y nvidia-cuda-toolkit)"},
	{Name: "chromium", Required: false, MinVersion: "", InstallCommand: "sudo pacman -S --noconfirm chromium || sudo apt install -y chromium-browser || sudo apt install -y chromium", UpgradeCommand: "sudo pacman -Syu --noconfirm chromium || (sudo apt update && sudo apt upgrade -y chromium-browser)"},
}

// LoadSystemConfig loads or creates system.json
func LoadSystemConfig() (*SystemConfig, error) {
	// Always perform a fresh scan to ensure data is accurate
	sys, err := IntrospectSystem()
	if err != nil {
		return nil, err
	}

	// Save the fresh scan
	_ = SaveSystemConfig(sys)

	return sys, nil
}

// IntrospectSystem scans hardware and software

func IntrospectSystem() (*SystemConfig, error) {

	sys := &SystemConfig{

		CPU: detectCPU(),

		GPU: detectGPU(),

		Storage: detectStorage(),

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

		Label: "Unknown CPU",

		Count: cores,

		Threads: threads,

		AvgGHz: 0,

		MaxGHz: 0,
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

				// Fix: Trim space before splitting or parsing to handle " 4096 MiB" correctly

				vramPart := strings.TrimSpace(parts[1])

				vramStr := strings.Fields(vramPart)[0]

				vram, _ := strconv.ParseInt(vramStr, 10, 64)

				vram *= 1024 * 1024 // Convert MB to bytes

				gpus = append(gpus, GPUInfo{

					Label: name,

					CUDA: 1, // Assume CUDA available if nvidia-smi works

					VRAM: vram,

					ComputePriority: i,

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

type LSBLKOutput struct {
	BlockDevices []BlockDevice `json:"blockdevices"`
}

type BlockDevice struct {
	Name string `json:"name"`

	Size int64 `json:"size"`

	FSUsed *int64 `json:"fsused"`

	Type string `json:"type"`

	MountPoint *string `json:"mountpoint"`

	Children []BlockDevice `json:"children"`
}

// detectStorage lists storage devices with usage information

func detectStorage() []StorageInfo {

	var storage []StorageInfo

	// Try lsblk on Linux

	if runtime.GOOS != "linux" {

		return storage

	}

	// Use -J for JSON output, -b for bytes

	cmd := exec.Command("lsblk", "-J", "-b", "-o", "NAME,SIZE,FSUSED,TYPE,MOUNTPOINT")

	output, err := cmd.Output()

	if err != nil {

		return storage

	}

	var lsblkOut LSBLKOutput

	if err := json.Unmarshal(output, &lsblkOut); err != nil {

		return storage

	}

	for _, device := range lsblkOut.BlockDevices {

		// We only care about physical disks

		if device.Type != "disk" {

			continue

		}

		info := StorageInfo{

			Device: device.Name,

			Size: device.Size,

			Used: 0,

			MountPoint: "unmounted",
		}

		// Calculate total used from children (partitions)

		var totalUsed int64

		var bestMountPoint string

		var scanChildren func([]BlockDevice)

		scanChildren = func(children []BlockDevice) {

			for _, child := range children {

				if child.FSUsed != nil {

					totalUsed += *child.FSUsed

				}

				if child.MountPoint != nil && *child.MountPoint != "" {

					// Prioritize root, then shortest path

					if *child.MountPoint == "/" {

						bestMountPoint = "/"

					} else if bestMountPoint != "/" {

						if bestMountPoint == "" || len(*child.MountPoint) < len(bestMountPoint) {

							bestMountPoint = *child.MountPoint

						}

					}

				}

				if len(child.Children) > 0 {

					scanChildren(child.Children)

				}

			}

		}

		scanChildren(device.Children)

		info.Used = totalUsed

		if bestMountPoint != "" {

			info.MountPoint = bestMountPoint

		}

		storage = append(storage, info)

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
		"chromium":      {"--version"},
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

	// Special handling for git
	if pkgName == "git" {
		// Output: git version 2.52.0
		parts := strings.Fields(versionStr)
		if len(parts) >= 3 && parts[0] == "git" && parts[1] == "version" {
			return parts[2]
		}
	}

	// Special handling for go
	if pkgName == "go" {
		// Output: go version go1.25.5 ...
		parts := strings.Fields(versionStr)
		if len(parts) >= 3 && parts[0] == "go" && parts[1] == "version" {
			return strings.TrimPrefix(parts[2], "go")
		}
	}

	// Special handling for python3
	if pkgName == "python3" {
		// Output: Python 3.13.11
		parts := strings.Fields(versionStr)
		if len(parts) >= 2 && parts[0] == "Python" {
			return parts[1]
		}
	}

	// Special handling for redis-server
	if pkgName == "redis-server" {
		// Output: Redis server v=8.4.0 ...
		parts := strings.Fields(versionStr)
		for _, part := range parts {
			if strings.HasPrefix(part, "v=") {
				return strings.TrimPrefix(part, "v=")
			}
		}
	}

	// Special handling for htop
	if pkgName == "htop" {
		// Output: htop 3.4.1-3.4.1
		parts := strings.Fields(versionStr)
		if len(parts) >= 2 && parts[0] == "htop" {
			return strings.Split(parts[1], "-")[0]
		}
	}

	// Special handling for lsblk / findmnt
	if pkgName == "lsblk" || pkgName == "findmnt" {
		// Output: lsblk from util-linux 2.41.3
		parts := strings.Fields(versionStr)
		if len(parts) >= 4 && parts[2] == "util-linux" {
			return parts[3]
		}
	}

	// Special handling for jq
	if pkgName == "jq" {
		// Output: jq-1.8.1
		return strings.TrimPrefix(versionStr, "jq-")
	}

	// Special handling for chromium
	if pkgName == "chromium" {
		// Output: Chromium 143.0.7499.169 Arch Linux
		parts := strings.Fields(versionStr)
		if len(parts) >= 2 && parts[0] == "Chromium" {
			return parts[1]
		}
	}

	// Special handling for nvidia-smi to extract just the driver version
	if pkgName == "nvidia-smi" {
		lines := strings.Split(versionStr, "\n")
		for _, line := range lines {
			if strings.Contains(line, "DRIVER version") || strings.Contains(line, "Driver Version") {
				parts := strings.Split(line, ":")
				if len(parts) >= 2 {
					return strings.TrimSpace(parts[1])
				}
			}
		}
		// Fallback if specific line not found, but try to be cleaner
		if len(lines) > 0 {
			return lines[0]
		}
	}

	// Special handling for nvcc
	if pkgName == "nvcc" {
		lines := strings.Split(versionStr, "\n")
		for _, line := range lines {
			if strings.Contains(line, "release") {
				parts := strings.Split(line, ",")
				for _, part := range parts {
					if strings.Contains(part, "V") {
						return strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(part), "V"))
					}
				}
			}
		}
	}

	// Special handling for pip to extract just the version number
	if pkgName == "pip3" || pkgName == "pip" {
		// Output: pip 25.3 from /path/to/site-packages/pip (python 3.13)
		parts := strings.Fields(versionStr)
		if len(parts) >= 2 && parts[0] == "pip" {
			return parts[1]
		}
	}

	// Special handling for curl
	if pkgName == "curl" {
		// Output: curl 8.17.0 (x86_64-pc-linux-gnu) ...
		parts := strings.Fields(versionStr)
		if len(parts) >= 2 && parts[0] == "curl" {
			return parts[1]
		}
	}

	// Special handling for make
	if pkgName == "make" {
		// Output: GNU Make 4.4.1 ...
		parts := strings.Fields(versionStr)
		if len(parts) >= 3 && parts[0] == "GNU" && parts[1] == "Make" {
			return parts[2]
		}
	}

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
