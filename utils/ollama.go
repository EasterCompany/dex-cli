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

const DefaultOllamaURL = "http://localhost:11434"

var DefaultModels = []string{"llama2", "mistral"}

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

	client := &http.Client{Timeout: 30 * time.Second}
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

	ui.PrintInfo(fmt.Sprintf("Starting pull for model: %s...", modelID))

	for {
		line, err := reader.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading pull response stream: %w", err)
		}
		if strings.TrimSpace(line) == "" {
			continue
		}

		var chunk PullResponse
		if jsonErr := json.Unmarshal([]byte(line), &chunk); jsonErr != nil {
			ui.PrintInfo(fmt.Sprintf("Error unmarshalling chunk: %v. Raw: %s", jsonErr, line))
			continue
		}

		if chunk.Total > 0 && chunk.Completed > 0 {
			percent := float64(chunk.Completed) / float64(chunk.Total) * 100

			label := fmt.Sprintf("%s: [%s] (%s of %s)",
				chunk.Status,
				chunk.Digest,
				formatBytes(chunk.Completed),
				formatBytes(chunk.Total),
			)
			ui.PrintProgressBar(label, int(percent))

		} else {
			ui.PrintInfo(fmt.Sprintf("%s: %s", chunk.Status, modelID))
		}
	}

	ui.PrintInfo("Model pull completed.")
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

// PullHardcodedModels iterates over a hardcoded list of models and pulls them.
func PullHardcodedModels() error {
	var errs []error
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
		// FIX: Use constant format string to satisfy govet.
		return fmt.Errorf("%s", sb.String())
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
		// FIX: Use keyed fields for ui.KeyVal struct literal.
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

	// FIX: Use keyed fields for ui.KeyVal struct literal.
	ui.PrintKeyValBlock("OLLAMA SERVICE STATUS", []ui.KeyVal{
		{Key: "Status", Value: status},
		{Key: "URL", Value: url},
		{Key: "Message", Value: message},
	})

	return err
}
