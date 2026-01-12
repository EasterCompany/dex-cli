package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/EasterCompany/dex-cli/cache"
	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/ui"
	"github.com/EasterCompany/dex-cli/utils"
)

func Agent(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: dex agent <name> <command> [flags]")
	}

	agentName := args[0]
	command := args[1]
	force := false

	// Check flags starting from index 2
	for _, arg := range args[2:] {
		if arg == "-f" || arg == "--force" {
			force = true
		}
	}

	def, err := config.Resolve("event")
	if err != nil {
		return fmt.Errorf("failed to resolve event service: %w", err)
	}

	switch agentName {
	case "guardian":
		return handleGuardian(def, command, force)
	case "analyst", "analyzer": // Alias for flexibility
		return handleAnalyst(def, command, force)
	case "imaginator":
		return handleImaginator(def, command, force)
	default:
		return fmt.Errorf("unknown agent: %s. Available agents: guardian, analyzer, imaginator", agentName)
	}
}

func handleGuardian(def *config.ServiceDefinition, command string, force bool) error {
	switch command {
	case "run":
		ui.PrintHeader("Guardian Agent")

		// 1. Wait for system idle and no ongoing processes
		if !force {
			ui.PrintRunningStatus("Verifying system state...")

			// Setup Redis for queue registration
			ctx := context.Background()
			redisClient, _ := cache.GetLocalClient(ctx)
			if redisClient != nil {
				defer func() {
					_ = redisClient.Del(ctx, "process:queued:system-guardian").Err()
					_ = redisClient.Close()
				}()
			}

			for {
				if redisClient != nil {
					queueInfo := map[string]interface{}{
						"channel_id": "system-guardian",
						"state":      "Waiting...",
						"start_time": time.Now().Unix(),
						"pid":        os.Getpid(),
						"updated_at": time.Now().Unix(),
					}
					qBytes, _ := json.Marshal(queueInfo)
					_ = redisClient.Set(ctx, "process:queued:system-guardian", qBytes, 15*time.Second).Err()
				}

				// Check busy processes (busy_ref_count > 0)
				statusBody, _, err := utils.GetHTTPBody(def.GetHTTP("/processes"))
				if err == nil {
					var procData struct {
						Active []interface{} `json:"active"`
					}
					if json.Unmarshal(statusBody, &procData) == nil && len(procData.Active) == 0 {
						// Check system idle time via agent status
						agentStatusBody, _, err := utils.GetHTTPBody(def.GetHTTP("/agent/status"))
						if err == nil {
							var status struct {
								Agents struct {
									Guardian struct {
										Protocols struct {
											Sentry struct {
												NextRun int64 `json:"next_run"`
											} `json:"sentry"`
											Architect struct {
												NextRun int64 `json:"next_run"`
											} `json:"architect"`
										} `json:"protocols"`
									} `json:"guardian"`
								} `json:"agents"`
								System struct {
									State     string `json:"state"`
									StateTime int64  `json:"state_time"`
								} `json:"system"`
							}

							if json.Unmarshal(agentStatusBody, &status) == nil {
								now := time.Now().Unix()
								t1Next := status.Agents.Guardian.Protocols.Sentry.NextRun
								t2Next := status.Agents.Guardian.Protocols.Architect.NextRun

								t1Ready := now >= t1Next
								t2Ready := now >= t2Next

								idleSecs := int64(0)
								if status.System.State == "idle" {
									idleSecs = status.System.StateTime
								}

								idleReady := idleSecs >= 300

								if idleReady && t1Ready && t2Ready {
									break // All clear
								} else {
									ui.PrintRunningStatus(fmt.Sprintf("Waiting for cooldown/idle... (Idle: %ds, Sentry Ready: %v, Architect Ready: %v)", idleSecs, t1Ready, t2Ready))
								}
							}
						}
					} else {
						ui.PrintRunningStatus(fmt.Sprintf("System busy with %d active processes. Waiting...", len(procData.Active)))
					}
				}
				time.Sleep(10 * time.Second)
			}
		}

		ui.PrintInfo("Triggering Full Analysis (Sentry + Architect)...")

		// 2. Trigger analysis via Event Service
		url := fmt.Sprintf("%s?tier=0", def.GetHTTP("/guardian/run"))
		req, _ := http.NewRequest("POST", url, nil)
		client := &http.Client{Timeout: 15 * time.Minute}

		ui.PrintRunningStatus("Executing Guardian protocols...")
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("failed to trigger guardian: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("guardian run failed with status: %d", resp.StatusCode)
		}

		ui.PrintSuccess("Guardian run completed successfully.")

	case "reset":
		ui.PrintHeader("Guardian Reset")
		ui.PrintInfo("Resetting Guardian protocols...")

		url := fmt.Sprintf("%s?tier=all", def.GetHTTP("/guardian/reset"))
		req, _ := http.NewRequest("POST", url, nil)
		client := &http.Client{Timeout: 10 * time.Second}

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("failed to reset guardian: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("guardian reset failed with status: %d", resp.StatusCode)
		}

		ui.PrintSuccess("Guardian protocols reset successfully.")

	default:
		return fmt.Errorf("unknown guardian command: %s", command)
	}
	return nil
}

func handleAnalyst(def *config.ServiceDefinition, command string, force bool) error {
	switch command {
	case "run":
		ui.PrintHeader("Analyst Agent")
		ui.PrintInfo("Triggering Synthesis Protocol...")

		url := def.GetHTTP("/analyzer/run")
		req, _ := http.NewRequest("POST", url, nil)
		client := &http.Client{Timeout: 10 * time.Second}

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("failed to trigger analyst: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("analyst run failed with status: %d", resp.StatusCode)
		}

		ui.PrintSuccess("Analyst synthesis protocol triggered successfully in background.")
		ui.PrintInfo("You can monitor progress in the dashboard or service logs.")

	case "reset":
		ui.PrintHeader("Analyst Reset")
		ui.PrintInfo("Resetting Analyst protocols...")

		url := def.GetHTTP("/analyzer/reset")
		req, _ := http.NewRequest("POST", url, nil)
		client := &http.Client{Timeout: 10 * time.Second}

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("failed to reset analyst: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("analyst reset failed with status: %d", resp.StatusCode)
		}

		ui.PrintSuccess("Analyst protocols reset successfully.")

	default:
		return fmt.Errorf("unknown analyst command: %s", command)
	}
	return nil
}

func handleImaginator(def *config.ServiceDefinition, command string, force bool) error {
	switch command {
	case "run":
		ui.PrintHeader("Imaginator Agent")
		ui.PrintInfo("Triggering Alert Review Protocol...")

		// Trigger Alert Review via Guardian endpoint (shared logic)
		url := fmt.Sprintf("%s?tier=alert_review", def.GetHTTP("/guardian/run"))
		req, _ := http.NewRequest("POST", url, nil)
		client := &http.Client{Timeout: 5 * time.Minute}

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("failed to trigger imaginator: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("imaginator run failed with status: %d", resp.StatusCode)
		}

		ui.PrintSuccess("Imaginator alert review protocol triggered successfully.")

	case "reset":
		ui.PrintHeader("Imaginator Reset")
		ui.PrintInfo("Resetting Imaginator protocols...")

		url := fmt.Sprintf("%s?tier=alert_review", def.GetHTTP("/guardian/reset"))
		req, _ := http.NewRequest("POST", url, nil)
		client := &http.Client{Timeout: 10 * time.Second}

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("failed to reset imaginator: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("imaginator reset failed with status: %d", resp.StatusCode)
		}

		ui.PrintSuccess("Imaginator protocols reset successfully.")

	default:
		return fmt.Errorf("unknown imaginator command: %s", command)
	}
	return nil
}
