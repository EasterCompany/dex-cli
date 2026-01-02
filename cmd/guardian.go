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

func Guardian(args []string) error {
	command := "run"
	force := false

	for _, arg := range args {
		switch arg {
		case "-f", "--force":
			force = true
		case "run", "reset":
			command = arg
		}
	}

	def, err := config.Resolve("event")
	if err != nil {
		return fmt.Errorf("failed to resolve event service: %w", err)
	}

	switch command {
	case "run":
		ui.PrintHeader("Guardian Technical Sentry")

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
						// Check system idle time via guardian status
						guardianStatusBody, _, err := utils.GetHTTPBody(def.GetHTTP("/guardian/status"))
						if err == nil {
							var gs struct {
								SystemIdleTime int64 `json:"system_idle_time"`
							}
							if json.Unmarshal(guardianStatusBody, &gs) == nil {
								// Check if all protocols are off cooldown
								var gsFull struct {
									Tier1 struct {
										LastRun int64 `json:"last_run"`
									} `json:"t1"`
									Tier2 struct {
										LastRun int64 `json:"last_run"`
									} `json:"t2"`
									SystemIdleTime int64 `json:"system_idle_time"`
								}
								_ = json.Unmarshal(guardianStatusBody, &gsFull)

								now := time.Now().Unix()
								t1Ready := (now - gsFull.Tier1.LastRun) >= 1800
								t2Ready := (now - gsFull.Tier2.LastRun) >= 1800
								idleReady := gs.SystemIdleTime >= 300

								if idleReady && t1Ready && t2Ready {
									break // All clear
								} else {
									ui.PrintRunningStatus(fmt.Sprintf("Waiting for cooldown/idle... (Idle: %ds, Sentry Ready: %v, Architect Ready: %v)", gs.SystemIdleTime, t1Ready, t2Ready))
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

		// 2. Trigger analysis via Event Service (Default to tier 0)
		url := fmt.Sprintf("%s?tier=0", def.GetHTTP("/guardian/run"))
		req, _ := http.NewRequest("POST", url, nil)
		client := &http.Client{Timeout: 15 * time.Minute} // Analysis + Tests can take a while

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

		url := def.GetHTTP("/guardian/reset")
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
