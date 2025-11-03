package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// OptionsConfig represents the structure of options.json
type OptionsConfig struct {
	Doc     string `json:"_doc"`
	Editor  string `json:"editor"`
	Theme   string `json:"theme"`
	Logging bool   `json:"logging"`
}

// DefaultOptionsConfig returns the default options configuration
func DefaultOptionsConfig() *OptionsConfig {
	return &OptionsConfig{
		Doc:     "User-specific options for dex-cli",
		Editor:  "vscode",
		Theme:   "dark",
		Logging: true,
	}
}

// LoadOptionsConfig loads the options.json file
func LoadOptionsConfig() (*OptionsConfig, error) {
	optionsPath, err := ExpandPath(filepath.Join(DexterRoot, "config", "options.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to expand options.json path: %w", err)
	}

	data, err := os.ReadFile(optionsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, os.ErrNotExist
		}
		return nil, fmt.Errorf("failed to read options.json: %w", err)
	}

	var options OptionsConfig
	if err := json.Unmarshal(data, &options); err != nil {
		return nil, fmt.Errorf("failed to parse options.json: %w", err)
	}

	return &options, nil
}

// SaveOptionsConfig saves the options.json file
func SaveOptionsConfig(options *OptionsConfig) error {
	optionsPath, err := ExpandPath(filepath.Join(DexterRoot, "config", "options.json"))
	if err != nil {
		return fmt.Errorf("failed to expand options.json path: %w", err)
	}

	data, err := json.MarshalIndent(options, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal options.json: %w", err)
	}

	return os.WriteFile(optionsPath, data, 0644)
}
