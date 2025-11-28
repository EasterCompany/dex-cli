package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// OptionsConfig represents the structure of options.json
type OptionsConfig struct {
	Doc     string         `json:"_doc"`
	Editor  string         `json:"editor"`
	Theme   string         `json:"theme"`
	Logging bool           `json:"logging"`
	Discord DiscordOptions `json:"discord"`
}

// DiscordOptions holds discord specific configurations
type DiscordOptions struct {
	Token               string `json:"token"`
	ServerID            string `json:"server_id"`
	DebugChannelID      string `json:"debug_channel_id"`
	MasterUser          string `json:"master_user"`
	DefaultVoiceChannel string `json:"default_voice_channel"`
}

// DefaultOptionsConfig returns the default options configuration
func DefaultOptionsConfig() *OptionsConfig {
	return &OptionsConfig{
		Doc:     "User-specific options for dex-cli and related services",
		Editor:  "vscode",
		Theme:   "dark",
		Logging: true,
		Discord: DiscordOptions{
			Token:               "YOUR_DISCORD_BOT_TOKEN_HERE",
			ServerID:            "YOUR_DISCORD_SERVER_ID_HERE",
			DebugChannelID:      "YOUR_DISCORD_DEBUG_CHANNEL_ID_HERE",
			MasterUser:          "313071000877137920",
			DefaultVoiceChannel: "1427777517414125578",
		},
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

	return os.WriteFile(optionsPath, data, 0o644)
}
