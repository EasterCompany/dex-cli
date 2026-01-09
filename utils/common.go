package utils

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/EasterCompany/dex-cli/cache"
	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/ui"
)

// WipeRedis clears the entire Redis database.
func WipeRedis(ctx context.Context) error {
	ui.PrintInfo("Wiping Redis database (clearing runtime state)...")
	redisClient, err := cache.GetLocalClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}
	defer func() { _ = redisClient.Close() }()

	if err := redisClient.FlushAll(ctx).Err(); err != nil {
		return fmt.Errorf("failed to wipe Redis: %w", err)
	}

	ui.PrintSuccess("Redis database wiped.")
	return nil
}

// GetConfiguredServices loads the service-map.json and merges its values
// with the master service definitions. This ensures user-configured
// domains, ports, and credentials are used.
func GetConfiguredServices() ([]config.ServiceDefinition, error) {
	// 1. Load the user's service-map.json
	serviceMap, err := config.LoadServiceMapConfig()
	if err != nil {
		if os.IsNotExist(err) {
			// If it doesn't exist, use the default map
			serviceMap = config.DefaultServiceMapConfig()
		} else {
			// Other error
			return nil, fmt.Errorf("failed to load service-map.json: %w", err)
		}
	}

	// 2. Get the hardcoded master list of all *possible* services
	// This master list knows the ShortName, Type, ID, etc.
	masterList := config.GetAllServices()
	masterDefs := make(map[string]config.ServiceDefinition)
	for _, def := range masterList {
		masterDefs[def.ID] = def
	}

	// 3. Create the final list by merging the service map values
	var configuredServices []config.ServiceDefinition

	for serviceType, serviceEntries := range serviceMap.Services {
		for _, entry := range serviceEntries {
			// Find the master definition for this service ID
			masterDef, ok := masterDefs[entry.ID]
			if !ok {
				// This service is in service-map.json but not in the hardcoded master list
				// We can't check it if we don't know its type, shortname etc.
				// A better approach: The masterDef *must* exist.
				// Let's assume for status, we only check services known to the CLI.
				continue
			}

			// Merge: Use master def as base, but override with user's config
			masterDef.Type = serviceType // Ensure type is from the map key
			if entry.Domain != "" {
				masterDef.Domain = entry.Domain
			}
			if entry.Port != "" {
				masterDef.Port = entry.Port
			}
			if entry.Credentials != nil {
				masterDef.Credentials = entry.Credentials
			}

			configuredServices = append(configuredServices, masterDef)
		}
	}

	// Sort by port to maintain a consistent order
	sort.Slice(configuredServices, func(i, j int) bool {
		return configuredServices[i].Port < configuredServices[j].Port
	})

	return configuredServices, nil
}

// HasArtifacts checks if a service has any artifacts that need to be backed up.
func HasArtifacts(service config.ServiceDefinition) bool {
	return service.Backup != nil && len(service.Backup.Artifacts) > 0
}

// ParseNumericInput parses user input to select services.
func ParseNumericInput(input string, count int) ([]int, error) {
	var selected []int
	parts := strings.Split(input, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.Contains(part, "-") {
			rangeParts := strings.Split(part, "-")
			if len(rangeParts) != 2 {
				return nil, fmt.Errorf("invalid range: %s", part)
			}
			start, err := strconv.Atoi(rangeParts[0])
			if err != nil {
				return nil, fmt.Errorf("invalid number: %s", rangeParts[0])
			}
			end, err := strconv.Atoi(rangeParts[1])
			if err != nil {
				return nil, fmt.Errorf("invalid number: %s", rangeParts[1])
			}
			if start > end {
				return nil, fmt.Errorf("invalid range: start > end")
			}
			for i := start; i <= end; i++ {
				if i > 0 && i <= count {
					selected = append(selected, i)
				}
			}
		} else {
			num, err := strconv.Atoi(part)
			if err != nil {
				return nil, fmt.Errorf("invalid number: %s", part)
			}
			if num > 0 && num <= count {
				selected = append(selected, num)
			}
		}
	}
	return selected, nil
}

// FormatBytes converts bytes to a human-readable string.
func FormatBytes(b int64) string {
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

// GenerateRandomHash creates a random lowercase letter string of a given length.
func GenerateRandomHash(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz"
	seededRand := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}
