// cache/client.go
package cache

import (
	"context"
	"fmt"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/redis/go-redis/v9"
)

// GetLocalClient finds the 'local-cache-0' service from the service map
// and returns an initialized Redis client.
func GetLocalClient(ctx context.Context) (*redis.Client, error) {
	// 1. Load the user's service map
	serviceMap, err := config.LoadServiceMapConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load service-map.json: %w", err)
	}

	// 2. Find the 'local-cache-0' definition
	var cacheDef *config.ServiceEntry
	if osServices, ok := serviceMap.Services["os"]; ok {
		for i, s := range osServices {
			if s.ID == "local-cache-0" {
				cacheDef = &osServices[i]
				break
			}
		}
	}

	if cacheDef == nil {
		// Fallback to hardcoded default if not in service map (should not happen)
		def, err := config.Resolve("local-cache")
		if err != nil {
			return nil, fmt.Errorf("local-cache-0 service definition not found")
		}
		entry := def.ToServiceEntry()
		cacheDef = &entry
	}

	// 3. Create Redis client options
	opts := &redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cacheDef.Domain, cacheDef.Port),
		Password: "", // default
		DB:       0,  // default
	}

	if cacheDef.Credentials != nil {
		opts.Password = cacheDef.Credentials.Password
		opts.DB = cacheDef.Credentials.DB
		// Note: go-redis doesn't use Username for non-ACL connections
	}

	// 4. Create and test client connection
	client := redis.NewClient(opts)
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping local cache at %s: %w", opts.Addr, err)
	}

	return client, nil
}
