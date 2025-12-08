package utils

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"
	"time"

	"github.com/EasterCompany/dex-cli/config"
)

// InstallSystemdService installs a systemd service for the given definition.
// It handles standard binaries and special cases like python scripts or static sites.
func InstallSystemdService(service config.ServiceDefinition) error {
	systemdDir := os.ExpandEnv("$HOME/.config/systemd/user")
	if err := os.MkdirAll(systemdDir, 0755); err != nil {
		return fmt.Errorf("failed to create systemd directory: %w", err)
	}

	servicePath := filepath.Join(systemdDir, service.SystemdName)

	// Prepare template data
	type ServiceData struct {
		Description string
		ExecStart   string
		WorkingDir  string
		Environment string
		LogPath     string
	}

	data := ServiceData{
		Description: fmt.Sprintf("Dexter Service: %s", service.ShortName),
		LogPath:     fmt.Sprintf("%%h/Dexter/logs/%s.log", service.ID),
	}

	// Set WorkingDir to the service's source path if available
	if service.Source != "" {
		expandedSourcePath, err := config.ExpandPath(service.Source)
		if err == nil {
			data.WorkingDir = expandedSourcePath
		}
	} else {
		data.WorkingDir = os.ExpandEnv("$HOME") // Fallback for services without a source dir
	}

	// Determine ExecStart based on service type
	switch service.Type {
	case "fe": // Frontend (served via dex serve)
		dexPath := os.ExpandEnv("$HOME/Dexter/bin/dex")
		sourcePath, _ := config.ExpandPath(service.Source)

		// Serve directly from source root (GitHub Pages style)
		data.ExecStart = fmt.Sprintf("%s serve --dir %s --port %s", dexPath, sourcePath, service.Port)

	case "be": // Backend (Python or other)
		sourcePath, _ := config.ExpandPath(service.Source)

		// Check for run.sh which handles venv and everything
		runScript := filepath.Join(sourcePath, "run.sh")
		if _, err := os.Stat(runScript); err == nil {
			data.ExecStart = runScript
			data.WorkingDir = sourcePath
		} else {
			// Fallback or other types?
			// Ideally backend services should have a run script or standard binary.
			// If it's a binary we built (like dex-event-service):
			binaryPath := filepath.Join(os.ExpandEnv("$HOME/Dexter/bin"), service.ID)
			data.ExecStart = binaryPath
		}

	default: // "cs", "th" etc - usually Go binaries
		binaryPath := filepath.Join(os.ExpandEnv("$HOME/Dexter/bin"), service.ID)
		data.ExecStart = binaryPath
	}

	// Special environment variables?
	// data.Environment = "VAR=value"

	tmpl := `[Unit]
Description={{.Description}}
After=network.target

[Service]
Type=simple
ExecStart={{.ExecStart}}
WorkingDirectory={{.WorkingDir}}
Restart=always
RestartSec=5
StandardOutput=append:{{.LogPath}}
StandardError=append:{{.LogPath}}
{{if .Environment}}Environment={{.Environment}}{{end}}

[Install]
WantedBy=default.target
`

	file, err := os.Create(servicePath)
	if err != nil {
		return fmt.Errorf("failed to create service file: %w", err)
	}
	defer func() { _ = file.Close() }()

	t := template.Must(template.New("service").Parse(tmpl))
	if err := t.Execute(file, data); err != nil {
		return fmt.Errorf("failed to write service template: %w", err)
	}

	// Reload systemd
	if err := exec.Command("systemctl", "--user", "daemon-reload").Run(); err != nil {
		return fmt.Errorf("failed to reload systemd daemon: %w", err)
	}

	// Enable service
	if err := exec.Command("systemctl", "--user", "enable", service.SystemdName).Run(); err != nil {
		return fmt.Errorf("failed to enable service: %w", err)
	}

	// Restart service
	if err := exec.Command("systemctl", "--user", "restart", service.SystemdName).Run(); err != nil {
		return fmt.Errorf("failed to restart service: %w", err)
	}

	return nil
}

// IsPortAvailable checks if a port is available on the given host.
func IsPortAvailable(host string, port string) bool {
	timeout := time.Second
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), timeout)
	if err != nil {
		return true
	}
	if conn != nil {
		_ = conn.Close()
		return false
	}
	return true
}
