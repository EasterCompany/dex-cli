package utils

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/EasterCompany/dex-cli/ui"
)

const DefaultOllamaURL = "http://127.0.0.1:11434"

var DefaultModels = []string{
	"gpt-oss:20b",
}

// ModelInfo reflects a single model entry returned by the /api/tags endpoint.
type ModelInfo struct {
	Name       string    `json:"name"`
	ModifiedAt time.Time `json:"modified_at"`
	Size       int64     `json:"size"`
	Digest     string    `json:"digest"`
	Details    struct {
		Format            string   `json:"format"`
		Family            string   `json:"family"`
		Families          []string `json:"families"`
		ParameterSize     string   `json:"parameter_size"`
		QuantizationLevel string   `json:"quantization_level"`
	} `json:"details"`
}

// ListModelsResponse handles the full JSON response from /api/tags.
type ListModelsResponse struct {
	Models []ModelInfo `json:"models"`
}

// PullRequest is the JSON body sent to /api/pull.
type PullRequest struct {
	Name     string `json:"name"`
	Insecure bool   `json:"insecure"`
}

// PullResponse handles a single chunk from the streaming response of /api/pull.
type PullResponse struct {
	Status    string `json:"status"`
	Digest    string `json:"digest,omitempty"`
	Total     int64  `json:"total,omitempty"`
	Completed int64  `json:"completed,omitempty"`
}

// GenerateRequest is the JSON body sent to /api/generate for non-streaming.
type GenerateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

// GenerateResponse handles the JSON response from /api/generate.
type GenerateResponse struct {
	Model     string    `json:"model"`
	CreatedAt time.Time `json:"created_at"`
	Response  string    `json:"response"`
	Done      bool      `json:"done"`
}

func doOllamaRequest(method, endpoint string, reqBody interface{}) ([]byte, error) {
	url := DefaultOllamaURL + endpoint

	var bodyReader io.Reader
	if reqBody != nil {
		reqBytes, err := json.Marshal(reqBody)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewBuffer(reqBytes)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{Timeout: 300 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ollama at %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama API request failed (status %d): %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	return io.ReadAll(resp.Body)
}

// ListModelsFull retrieves all available models with their complete details.
func ListModelsFull() ([]ModelInfo, error) {
	data, err := doOllamaRequest(http.MethodGet, "/api/tags", nil)
	if err != nil {
		return nil, err
	}

	var response ListModelsResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal model list response: %w", err)
	}

	return response.Models, nil
}

// ListModelIDs retrieves only the identifiers (e.g., "llama2:7b") for all available models.
func ListModelIDs() ([]string, error) {
	models, err := ListModelsFull()
	if err != nil {
		return nil, err
	}

	ids := make([]string, len(models))
	for i, m := range models {
		ids[i] = m.Name
	}
	return ids, nil
}

// ListModelNames retrieves only the base names (e.g., "llama2") for all available models.
func ListModelNames() ([]string, error) {
	models, err := ListModelsFull()
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(models))
	seenNames := make(map[string]bool)

	for _, m := range models {
		parts := strings.SplitN(m.Name, ":", 2)
		baseName := parts[0]

		if _, seen := seenNames[baseName]; !seen {
			names = append(names, baseName)
			seenNames[baseName] = true
		}
	}
	return names, nil
}

// PullModel initiates a model download and prints progress to os.Stdout using the UI library.
// This function intentionally breaks the "no print" rule of the utility package to fulfill the progress requirement.
func PullModel(modelID string) error {
	url := DefaultOllamaURL + "/api/pull"
	reqBody := PullRequest{Name: modelID}

	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal pull request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(reqBytes))
	if err != nil {
		return fmt.Errorf("failed to create pull request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 0}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to Ollama at %s for pull: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ollama API pull failed (status %d): %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	reader := bufio.NewReader(resp.Body)

	var lastStatus string
	spinFrame := 0

	for {
		line, err := reader.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			ui.ClearLine()
			return fmt.Errorf("error reading pull response stream: %w", err)
		}
		if strings.TrimSpace(line) == "" {
			continue
		}

		var chunk PullResponse
		if jsonErr := json.Unmarshal([]byte(line), &chunk); jsonErr != nil {
			// Silently skip malformed chunks
			continue
		}

		// Update status
		if chunk.Status != "" {
			lastStatus = chunk.Status
		}

		if chunk.Total > 0 && chunk.Completed > 0 {
			// Show progress bar with download info
			percent := float64(chunk.Completed) / float64(chunk.Total) * 100
			label := fmt.Sprintf("Pulling %s (%s/%s)", modelID, formatBytes(chunk.Completed), formatBytes(chunk.Total))
			ui.PrintProgressBar(label, int(percent))
		} else {
			// Show spinner for status updates without progress
			label := fmt.Sprintf("%s: %s", lastStatus, modelID)
			ui.PrintSpinner(label, spinFrame)
			spinFrame++
		}
	}

	// Clear the line and print completion message
	ui.ClearLine()
	ui.PrintSuccess(fmt.Sprintf("Successfully pulled model: %s", modelID))
	return nil
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}

func PullHardcodedModels() error {
	var errs []error

	// Step 1: Pull base models
	ui.PrintInfo("Pulling base models...")
	for _, model := range DefaultModels {
		if err := PullModel(model); err != nil {
			errs = append(errs, fmt.Errorf("failed to pull model %s: %v", model, err))
		}
	}

	if len(errs) > 0 {
		var sb strings.Builder
		sb.WriteString("Failed to pull one or more hardcoded models:\n")
		for _, err := range errs {
			sb.WriteString("- ")
			sb.WriteString(err.Error())
			sb.WriteString("\n")
		}
		return fmt.Errorf("%s", sb.String())
	}

	// Step 2: Create custom forked models
	ui.PrintInfo("Creating custom Dexter models...")
	if err := CreateCustomModels(); err != nil {
		ui.PrintWarning(fmt.Sprintf("Failed to create custom models (non-fatal): %v", err))
	}

	return nil
}

func CleanupNonDefaultModels() error {
	models, err := ListModelIDs()
	if err != nil {
		return fmt.Errorf("failed to list models: %w", err)
	}

	// Create a map of default models for quick lookup
	defaultMap := make(map[string]bool)
	for _, model := range DefaultModels {
		defaultMap[model] = true
	}

	// Also keep our custom dex models
	var toDelete []string
	for _, model := range models {
		// Skip if it's a default model
		if defaultMap[model] {
			continue
		}
		// Skip if it's a dex- prefixed model (our custom models)
		if strings.HasPrefix(model, "dex-") {
			continue
		}
		toDelete = append(toDelete, model)
	}

	// Delete non-default models
	for _, model := range toDelete {
		ui.PrintInfo(fmt.Sprintf("  Removing model: %s", model))
		if err := DeleteModel(model); err != nil {
			ui.PrintWarning(fmt.Sprintf("  Failed to delete %s: %v", model, err))
		}
	}

	if len(toDelete) == 0 {
		ui.PrintInfo("  No models to clean up.")
	}

	return nil
}

func DeleteModel(modelID string) error {
	url := DefaultOllamaURL + "/api/delete"
	reqBody := map[string]string{"name": modelID}

	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal delete request: %w", err)
	}

	req, err := http.NewRequest(http.MethodDelete, url, bytes.NewBuffer(reqBytes))
	if err != nil {
		return fmt.Errorf("failed to create delete request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete model: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete failed (status %d): %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	return nil
}

type CustomModel struct {
	Name         string
	BaseModel    string
	SystemPrompt string
}

func CreateCustomModels() error {
	customModels := []CustomModel{
		{
			Name:      "dex-commit-model",
			BaseModel: "gpt-oss:20b",
			SystemPrompt: `You are a git commit message generator. Analyze the provided diff and generate a concise, one-line commit message.

Format: <type>: <description>
Types: add, update, remove, refactor, fix, docs, test, style, chore

Rules:
- Output ONLY the single-line commit message.
- DO NOT include any other text, reasoning, or explanations.
- The description must be under 72 characters.
- Be specific and concise.`,
		},
		{
			Name:      "dex-summary-model",
			BaseModel: "gpt-oss:20b",
			SystemPrompt: `
You are a specialized AI assistant for generating summaries out of large and small bodies of text.
You may only create text summaries.
Your task is to analyze a piece of text (various formats: message logs, poems, news article) and generate clear, concise, and meaningful summary.`,
		},
		{
			Name:      "dex-engagement-model",
			BaseModel: "gpt-oss:20b",
			SystemPrompt: `You are an engagement analyzer. Your task is to determine if the given text requires a response from an AI assistant named Dexter. 

Analyze the context and intent. 
- If the user is addressing Dexter, asking a question, or making a statement that warrants a reply, output 'TRUE'.
- If the user is talking to someone else, muttering to themselves, or the text is noise/irrelevant, output 'FALSE'.

Output ONLY 'TRUE' or 'FALSE'. Do not explain.`,
		},
	}

	for _, model := range customModels {
		// First, delete the existing custom model if it exists (to force rebuild)
		ui.PrintInfo(fmt.Sprintf("  Rebuilding %s from %s...", model.Name, model.BaseModel))
		_ = DeleteModel(model.Name) // Ignore error - model might not exist yet

		// Create fresh custom model
		if err := CreateModelFromBase(model.Name, model.BaseModel, model.SystemPrompt); err != nil {
			ui.PrintWarning(fmt.Sprintf("  Failed to create %s: %v", model.Name, err))
			continue
		}
		ui.PrintSuccess(fmt.Sprintf("  Created %s", model.Name))
	}

	return nil
}

// CreateModelFromBase creates a custom model from a base model using the Ollama API.
func CreateModelFromBase(customName, baseModel, systemPrompt string) error {
	// Use explicit API fields instead of a modelfile string
	url := DefaultOllamaURL + "/api/create"
	reqBody := map[string]interface{}{
		"name":   customName,
		"from":   baseModel,
		"system": systemPrompt,
		"stream": false,
	}

	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal create request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(reqBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create model: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("create failed (status %d): %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	// Read the streaming response (model creation sends progress updates)
	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading create response: %w", err)
		}
		// The response is JSON lines with status updates, but we'll just consume them
		// In the future, we could parse and display progress
		_ = line
	}

	return nil
}

// GenerateContent sends a prompt to a specified model and waits for the complete response.
// It uses the non-streaming mode of the /api/generate endpoint.
func GenerateContent(modelID, prompt string) (string, error) {
	reqBody := GenerateRequest{
		Model:  modelID,
		Prompt: prompt,
		Stream: false,
	}

	data, err := doOllamaRequest(http.MethodPost, "/api/generate", reqBody)
	if err != nil {
		return "", err
	}

	var response GenerateResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return "", fmt.Errorf("failed to unmarshal generate response: %w", err)
	}

	return response.Response, nil
}

// GetOllamaStatus checks if the Ollama service is reachable and prints the status using UI functions.
func GetOllamaStatus() error {
	url := DefaultOllamaURL
	req, err := http.NewRequest(http.MethodHead, url, nil)
	if err != nil {
		ui.PrintKeyValBlock("OLLAMA SERVICE STATUS", []ui.KeyVal{
			{Key: "Status", Value: "ERROR"},
			{Key: "Message", Value: fmt.Sprintf("Failed to create request: %v", err)},
		})
		return err
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)

	status := "OFFLINE"
	message := fmt.Sprintf("Could not reach service at %s", url)

	if err != nil {
	} else {
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode == http.StatusOK {
			status = "ONLINE"
			message = fmt.Sprintf("Service is running at %s", url)
		} else {
			status = "WARNING"
			message = fmt.Sprintf("Service reached at %s, but returned status code %d", url, resp.StatusCode)
		}
	}
	ui.PrintKeyValBlock("OLLAMA SERVICE STATUS", []ui.KeyVal{
		{Key: "Status", Value: status},
		{Key: "URL", Value: url},
		{Key: "Message", Value: message},
	})

	return err
}

func GenerateCommitMessage(diff string) string {
	if strings.TrimSpace(diff) == "" {
		return "chore: code clean up"
	}

	// Truncate diff if too long (use a larger portion of context window)
	const maxDiffLength = 6000
	if len(diff) > maxDiffLength {
		diff = diff[:maxDiffLength] + "\n...(truncated)"
	}

	commitMsg, err := GenerateContent("dex-commit-model", diff)
	if err != nil {
		fmt.Printf("LLM Generation Error: %v\n", err)
		return "chore: code clean up"
	}

	// Clean up the message
	finalMsg := strings.TrimSpace(commitMsg)

	// If the model wrapped it in quotes, remove them
	finalMsg = strings.Trim(finalMsg, "\"'")

	// If the model still used XML tags despite instructions, strip them
	startTag := "<answer>"
	endTag := "</answer>"
	if strings.Contains(finalMsg, startTag) && strings.Contains(finalMsg, endTag) {
		startIndex := strings.Index(finalMsg, startTag)
		endIndex := strings.Index(finalMsg, endTag)
		finalMsg = finalMsg[startIndex+len(startTag) : endIndex]
		finalMsg = strings.TrimSpace(finalMsg)
	}

	// Basic validation: Look for the colon separator "type: description"
	if finalMsg != "" {
		// Ensure it starts with a known type or looks like "type: "
		// If it doesn't contain a colon, it might just be a description
		if !strings.Contains(finalMsg, ":") {
			finalMsg = "update: " + finalMsg
		}

		// Limit length (standard git shortlog)
		if len(finalMsg) > 72 {
			finalMsg = finalMsg[:69] + "..."
		}
		return finalMsg
	}

	// Fallback
	return "chore: code clean up"
}
