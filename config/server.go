// config/server.go
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ServerMapConfig represents the structure of server-map.json
type ServerMapConfig struct {
	Servers map[string]Server `json:"servers"`
}

// Server represents a single server in the server map
type Server struct {
	Host string `json:"host"`
	User string `json:"user"`
	Key  string `json:"key"`
}

// DefaultServerMapConfig returns the default server map configuration
func DefaultServerMapConfig() *ServerMapConfig {
	return &ServerMapConfig{
		Servers: map[string]Server{
			"onyx": {
				Host: "192.168.1.100",
				User: "root",
				Key:  "~/.ssh/id_rsa",
			},
		},
	}
}

// LoadServerMapConfig loads the server-map.json file
func LoadServerMapConfig() (*ServerMapConfig, error) {
	serverMapPath, err := ExpandPath(filepath.Join(DexterRoot, "config", "server-map.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to expand server-map.json path: %w", err)
	}

	data, err := os.ReadFile(serverMapPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, os.ErrNotExist
		}
		return nil, fmt.Errorf("failed to read server-map.json: %w", err)
	}

	var serverMap ServerMapConfig
	if err := json.Unmarshal(data, &serverMap); err != nil {
		return nil, fmt.Errorf("failed to parse server-map.json: %w", err)
	}

	return &serverMap, nil
}

// SaveServerMapConfig saves the server-map.json file
func SaveServerMapConfig(serverMap *ServerMapConfig) error {
	serverMapPath, err := ExpandPath(filepath.Join(DexterRoot, "config", "server-map.json"))
	if err != nil {
		return fmt.Errorf("failed to expand server-map.json path: %w", err)
	}

	data, err := json.MarshalIndent(serverMap, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal server-map.json: %w", err)
	}

	return os.WriteFile(serverMapPath, data, 0o644)
}
