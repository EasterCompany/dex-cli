package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/ui"
	"github.com/EasterCompany/dex-cli/utils"
)

type Chore struct {
	ID                 string `json:"id"`
	OwnerID            string `json:"owner_id"`
	Status             string `json:"status"`
	Schedule           string `json:"schedule"`
	LastRun            int64  `json:"last_run"`
	NaturalInstruction string `json:"natural_instruction"`
	ExecutionPlan      struct {
		EntryURL        string `json:"entry_url"`
		SearchQuery     string `json:"search_query"`
		ExtractionFocus string `json:"extraction_focus"`
	} `json:"execution_plan"`
	Memory []string `json:"memory"`
}

type MetadataResponse struct {
	URL     string `json:"url"`
	Content string `json:"content"`
	Title   string `json:"title"`
	Error   string `json:"error"`
}

type AIChoreResult struct {
	Found   bool     `json:"found"`
	Items   []string `json:"items"` // IDs of found items to add to memory
	Summary string   `json:"summary"`
}

func Courier(args []string) error {
	ui.PrintHeader("Courier Protocol")
	ui.PrintRunningStatus("Checking for active chores...")

	eventDef, err := config.Resolve("event")
	if err != nil {
		return fmt.Errorf("failed to resolve event service: %w", err)
	}

	// Fetch Chores
	body, _, err := utils.GetHTTPBody(eventDef.GetHTTP("/chores"))
	if err != nil {
		return fmt.Errorf("failed to fetch chores: %w", err)
	}

	var chores []Chore
	if err := json.Unmarshal(body, &chores); err != nil {
		return fmt.Errorf("failed to parse chores: %w", err)
	}

	runCount := 0
	for _, chore := range chores {
		if chore.Status != "active" {
			continue
		}

		// Check Schedule
		interval := 6 * time.Hour // Default
		if strings.HasPrefix(chore.Schedule, "every_") {
			durStr := strings.TrimPrefix(chore.Schedule, "every_")
			if d, err := time.ParseDuration(durStr); err == nil {
				interval = d
			}
		}

		// Check if due
		if time.Since(time.Unix(chore.LastRun, 0)) < interval {
			continue
		}

		runCount++
		// De-reference eventDef here
		if err := executeChore(*eventDef, chore); err != nil {
			ui.PrintError(fmt.Sprintf("Failed to execute chore '%s': %v", chore.NaturalInstruction, err))
		}
	}

	if runCount == 0 {
		ui.PrintInfo("No chores due.")
	} else {
		ui.PrintSuccess(fmt.Sprintf("Completed %d chores.", runCount))
	}

	return nil
}

func executeChore(eventDef config.ServiceDefinition, chore Chore) error {
	ui.PrintInfo(fmt.Sprintf("Running chore: %s", chore.NaturalInstruction))

	// 1. Fetch Content via Web Service
	webDef, err := config.Resolve("web")
	if err != nil {
		return err
	}

	targetURL := chore.ExecutionPlan.EntryURL
	if targetURL == "" {
		// Fallback to DuckDuckGo HTML search if no URL is provided
		query := chore.NaturalInstruction
		if chore.ExecutionPlan.SearchQuery != "" {
			query = chore.ExecutionPlan.SearchQuery
		} else {
			// Refine query using LLM
			refined := refineSearchQuery(query)
			if refined != "" {
				query = refined
			}
		}
		targetURL = fmt.Sprintf("https://html.duckduckgo.com/html?q=%s", url.QueryEscape(query))
		ui.PrintInfo(fmt.Sprintf("No Entry URL. Search: %s (Query: %s)", targetURL, query))
	}

	ui.PrintRunningStatus(fmt.Sprintf("Fetching content from %s...", targetURL))
	metaURL := fmt.Sprintf("%s?url=%s", webDef.GetHTTP("/metadata"), url.QueryEscape(targetURL))
	metaBody, statusCode, err := utils.GetHTTPBody(metaURL)
	if err != nil {
		return fmt.Errorf("failed to fetch metadata: %w", err)
	}

	if statusCode != 200 {
		return fmt.Errorf("web service returned status %d: %s", statusCode, string(metaBody))
	}

	var meta MetadataResponse
	if err := json.Unmarshal(metaBody, &meta); err != nil {
		return fmt.Errorf("failed to parse metadata: %w", err)
	}

	if meta.Error != "" {
		return fmt.Errorf("web service error: %s", meta.Error)
	}

	// 2. Prepare Prompt
	// Truncate content to fit context window roughly (assuming ~4 chars per token, safe buffer)
	content := meta.Content
	if len(content) > 12000 {
		content = content[:12000] + "...(truncated)"
	}

	prompt := fmt.Sprintf(`You are an AI Courier Agent.
User Instruction: "%s"
Search Context: "%s"
Extraction Focus: "%s"

Here is the text content of the webpage:
"""
%s
"""

Your Task:
1. Scan the content for SPECIFIC items matching the user's instruction.
2. CRITICAL: IGNORE all advertisements, sponsored listings, navigation menus (Home, About), generic footers, and tracking links.
3. IGNORE any items represented by these IDs/Strings (Already Seen): %v.
4. FIND MULTIPLE ITEMS: If more than one relevant item exists, list as many as possible (up to 10). Do not stop at the first match.
5. Return a JSON object ONLY.

Format:
{
  "found": true/false,
  "items": ["unique_id_1", "unique_id_2"], // List of NEW unique identifiers (titles, links, or IDs) found.
  "summary": "Detailed message with bullet points and markdown links: \n- [Item Title](Item URL) - brief snippet\n- ..." // Detailed report including direct links to relevant findings.
}
If no new items are found, set "found": false and "items": [].

Output JSON ONLY. No markdown wrapper.`,
		chore.NaturalInstruction,
		chore.ExecutionPlan.SearchQuery,
		chore.ExecutionPlan.ExtractionFocus,
		content,
		chore.Memory,
	)

	// 3. Call LLM
	ui.PrintRunningStatus("Analyzing content with AI...")
	response, err := utils.GenerateContent("dex-scraper-model", prompt)
	if err != nil {
		return fmt.Errorf("llm error: %w", err)
	}

	// Clean response
	response = strings.TrimSpace(response)
	response = strings.Trim(response, "`")
	response = strings.TrimPrefix(response, "json")

	var result AIChoreResult
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		// Fallback: try to just check if it says "found": true
		if strings.Contains(response, "\"found\": true") || strings.Contains(response, "\"found\":true") {
			result.Found = true
			result.Summary = "Items found, but failed to parse strict details. Check the link."
		} else {
			return fmt.Errorf("failed to parse LLM response: %s", response)
		}
	}

	// 4. Handle Result
	if result.Found {
		ui.PrintSuccess(fmt.Sprintf("Chore found new items! Summary: %s", result.Summary))

		utils.SendEvent("system.notification.generated", map[string]interface{}{
			"title":    "Courier Delivery",
			"body":     fmt.Sprintf("%s\n%s", chore.NaturalInstruction, result.Summary),
			"category": "cognitive",
			"priority": "low",
		})

		// Send Discord Notification
		discordDef, err := config.Resolve("discord")
		if err == nil && discordDef != nil {
			msgContent := fmt.Sprintf("📦 **Courier Update**\n\nTask: *%s*\n\n%s\n\n[Source Link](%s)", chore.NaturalInstruction, result.Summary, targetURL)

			// Use POST /post with user_id
			postURL := discordDef.GetHTTP("/post")
			postBody := map[string]interface{}{
				"user_id": chore.OwnerID,
				"content": msgContent,
			}
			jsonBody, _ := json.Marshal(postBody)

			if _, _, err := utils.PostHTTP(postURL, jsonBody); err != nil {
				ui.PrintWarning(fmt.Sprintf("Failed to send DM: %v", err))
			} else {
				ui.PrintSuccess("Sent DM to owner.")
			}
		}

		// Update Memory
		if len(result.Items) > 0 {
			newMemory := append(chore.Memory, result.Items...)

			// Call POST /chores/{id}/run
			runURL := eventDef.GetHTTP(fmt.Sprintf("/chores/%s/run", chore.ID))
			payload := map[string]interface{}{
				"memory": newMemory,
			}
			jsonPayload, _ := json.Marshal(payload)
			if _, _, err := utils.PostHTTP(runURL, jsonPayload); err != nil {
				ui.PrintWarning(fmt.Sprintf("Failed to update chore memory: %v", err))
			}
		}
	} else {
		ui.PrintInfo("Nothing new found.")
		// Update LastRun only
		runURL := eventDef.GetHTTP(fmt.Sprintf("/chores/%s/run", chore.ID))
		if _, _, err := utils.PostHTTP(runURL, nil); err != nil {
			ui.PrintWarning(fmt.Sprintf("Failed to update chore last_run: %v", err))
		}
	}

	return nil
}

// refineSearchQuery uses a small LLM to convert a natural instruction into a search query.
func refineSearchQuery(instruction string) string {
	prompt := fmt.Sprintf("Convert this user request into an optimal web search query (3-10 words). Keep key qualifiers (e.g. 'sale', 'rent', 'news'). Output ONLY the query: '%s'", instruction)
	response, err := utils.GenerateContent("dex-fast-summary-model", prompt)
	if err != nil {
		return instruction // Fallback
	}
	return strings.TrimSpace(strings.Trim(strings.TrimSpace(response), "\"'"))
}
