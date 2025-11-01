package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/ui"
	"github.com/EasterCompany/dex-cli/health"
)

const ( // Constants for service management
	pidFileExtension = ".pid"
	logFileExtension = ".log"
)

// Service manages start, stop, and restart operations for Dexter services.
func Service(action, serviceName string) error {
	ui.PrintTitle(fmt.Sprintf("DEXTER SERVICE COMMAND: %s %s", strings.ToUpper(action), strings.ToUpper(serviceName)))

	// Load the service map
	ui.PrintSectionTitle("LOADING SERVICE MAP")
	serviceMap, err := config.LoadServiceMap()
	if err != nil {
		return fmt.Errorf("failed to load service map: %w", err)
	}
	ui.PrintSuccess("Service map loaded")

	// Find the service entry
	var serviceEntry *config.ServiceEntry
	for _, services := range serviceMap.Services {
		for _, s := range services {
			if s.ID == serviceName {
				serviceEntry = &s
				break
			}
		}
		if serviceEntry != nil {
			break
		}
	}

	if serviceEntry == nil {
		return fmt.Errorf("service '%s' not found in service-map.json", serviceName)
	}

	// Perform the action
	switch action {
	case "start":
		return startService(serviceEntry)
	case "stop":
		return stopService(serviceEntry)
	case "restart":
		return restartService(serviceEntry)
	default:
		return fmt.Errorf("unknown service action: %s", action)
	}
}

func getPidFilePath(serviceID string) (string, error) {
	runPath, err := config.ExpandPath(filepath.Join(config.DexterRoot, "run"))
	if err != nil {
		return "", err
	}
	return filepath.Join(runPath, serviceID+pidFileExtension), nil
}

func getLogFilePath(serviceID string) (string, error) {
	logPath, err := config.ExpandPath(filepath.Join(config.DexterRoot, "logs"))
	if err != nil {
		return "", err
	}
	return filepath.Join(logPath, serviceID+logFileExtension), nil
}

func readPid(serviceID string) (int, error) {
	pidFilePath, err := getPidFilePath(serviceID)
	if err != nil {
		return 0, err
	}

	data, err := ioutil.ReadFile(pidFilePath)
	if err != nil {
		return 0, fmt.Errorf("failed to read PID file %s: %w", pidFilePath, err)
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, fmt.Errorf("invalid PID in file %s: %w", pidFilePath, err)
	}
	return pid, nil
}

func writePid(serviceID string, pid int) error {
	pidFilePath, err := getPidFilePath(serviceID)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(pidFilePath, []byte(strconv.Itoa(pid)), 0644)
}

func removePidFile(serviceID string) error {
	pidFilePath, err := getPidFilePath(serviceID)
	if err != nil {
		return err
	}
	return os.Remove(pidFilePath)
}

func isProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// Sending signal 0 to a process checks its existence without killing it
	return process.Signal(syscall.Signal(0)) == nil
}

func verifyServiceStatus(service *config.ServiceEntry) error {
	if service.Addr == "" {
		return fmt.Errorf("service %s has no address configured to check status", service.ID)
	}

	statusURL := strings.TrimSuffix(service.Addr, "/") + "/status"
	resp, err := http.Get(statusURL)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", statusURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("service %s /status returned non-200 status: %d", service.ID, resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body from %s: %w", statusURL, err)
	}

	var statusResp health.StatusResponse
	if err := json.Unmarshal(body, &statusResp); err != nil {
		return fmt.Errorf("failed to parse status response from %s: %w", statusURL, err)
	}

	if statusResp.Status != "healthy" {
		return fmt.Errorf("service %s reported status: %s", service.ID, statusResp.Status)
	}

	return nil
}

func startService(service *config.ServiceEntry) error {
	ui.PrintSectionTitle(fmt.Sprintf("STARTING %s", service.ID))

	// Check if already running
	pid, err := readPid(service.ID)
	if err == nil && isProcessRunning(pid) {
		ui.PrintWarning(fmt.Sprintf("%s is already running with PID %d", service.ID, pid))
		return nil
	}

	// Get binary path
	dexterBinPath, err := config.ExpandPath(filepath.Join(config.DexterRoot, "bin"))
	if err != nil {
		return err
	}
	binaryPath := filepath.Join(dexterBinPath, service.ID)

	// Check if binary exists
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		return fmt.Errorf("service binary not found: %s. Please run 'dex build %s'", binaryPath, service.ID)
	}

	// Get log file path
	logFilePath, err := getLogFilePath(service.ID)
	if err != nil {
		return err
	}

	// Open log file for redirecting stdout/stderr
	logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file %s: %w", logFilePath, err)
	}
	defer logFile.Close()

	// Start as background process using nohup
	cmd := exec.Command("nohup", binaryPath)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start service %s: %w", service.ID, err)
	}

	// Write PID to file
	if err := writePid(service.ID, cmd.Process.Pid); err != nil {
		return fmt.Errorf("failed to write PID file for %s: %w", service.ID, err)
	}

	ui.PrintInfo(fmt.Sprintf("%s started with PID %d. Logging to %s", service.ID, cmd.Process.Pid, logFilePath))

	// Verify service started by checking /status endpoint
	ui.PrintInfo("Verifying service status...")
	for i := 0; i < 10; i++ { // Retry 10 times with 1-second delay
		if err := verifyServiceStatus(service); err == nil {
			ui.PrintSuccess(fmt.Sprintf("%s started and is healthy", service.ID))
			return nil
		}
		time.Sleep(1 * time.Second)
	}

	return fmt.Errorf("service %s failed to become healthy after startup", service.ID)
}

func stopService(service *config.ServiceEntry) error {
	ui.PrintSectionTitle(fmt.Sprintf("STOPPING %s", service.ID))

	pid, err := readPid(service.ID)
	if err != nil || !isProcessRunning(pid) {
		ui.PrintWarning(fmt.Sprintf("%s is not running or PID file is missing", service.ID))
		_ = removePidFile(service.ID) // Clean up stale PID file if any
		return nil
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process %d for %s: %w", pid, service.ID, err)
	}

	// Send SIGTERM
	ui.PrintInfo(fmt.Sprintf("Sending SIGTERM to %s (PID %d)", service.ID, pid))
	if err := process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to send SIGTERM to %s: %w", service.ID, err)
	}

	// Wait for process to exit, with a timeout
	done := make(chan error, 1)
	go func() {
		_, err := process.Wait()
		done <- err
	}()

	select {
	case <-time.After(5 * time.Second): // 5-second timeout
		ui.PrintWarning(fmt.Sprintf("%s (PID %d) did not terminate gracefully, sending SIGKILL", service.ID, pid))
		if err := process.Signal(syscall.SIGKILL); err != nil {
			return fmt.Errorf("failed to send SIGKILL to %s: %w", service.ID, err)
		}
		<-done // Wait for SIGKILL to take effect
	case err := <-done:
		if err != nil {
			ui.PrintWarning(fmt.Sprintf("%s (PID %d) terminated with error: %v", service.ID, pid, err))
		}
	}

	// Remove PID file
	if err := removePidFile(service.ID); err != nil {
		ui.PrintError(fmt.Sprintf("Failed to remove PID file for %s: %v", service.ID, err))
	}

	ui.PrintSuccess(fmt.Sprintf("%s stopped successfully", service.ID))
	return nil
}

func restartService(service *config.ServiceEntry) error {
	ui.PrintSectionTitle(fmt.Sprintf("RESTARTING %s", service.ID))

	// Attempt to stop the service first
	_ = stopService(service) // Ignore error, as it might not be running

	// Then start it
	return startService(service)
}
