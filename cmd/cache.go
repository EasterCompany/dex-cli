// cmd/cache.go
package cmd

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/EasterCompany/dex-cli/cache"
	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/ui"
	"github.com/redis/go-redis/v9"
)

// Cache manages interaction with the local-cache-0 service.
func Cache(args []string) error {
	if len(args) == 0 {
		return printCacheUsage()
	}

	subcommand := args[0]
	switch subcommand {
	case "clear":
		return cacheClear()
	case "list":
		return cacheList()
	case "help":
		return printCacheUsage()
	default:
		return fmt.Errorf("unknown cache subcommand: %s", subcommand)
	}
}

func printCacheUsage() error {
	ui.PrintInfo("dex cache <subcommand>")
	ui.PrintInfo("  clear    Clear all keys from the local cache")
	ui.PrintInfo("  list     List all keys grouped by service, with size")
	return nil
}

// cacheClear connects to local-cache-0 and runs FLUSHDB.
func cacheClear() error {
	ui.PrintInfo("Connecting to local-cache-0...")
	client, err := cache.GetLocalClient(context.Background())
	if err != nil {
		return fmt.Errorf("failed to connect to local cache: %w", err)
	}
	defer func() { _ = client.Close() }()

	ui.PrintInfo("Clearing all keys from database...")
	if err := client.FlushDB(context.Background()).Err(); err != nil {
		return fmt.Errorf("failed to clear cache: %w", err)
	}

	ui.PrintSuccess("Local cache cleared successfully.")
	return nil
}

// cacheList scans all keys, groups them by service prefix, and reports size.
func cacheList() error {
	ui.PrintInfo("Connecting to local-cache-0 to scan keys...")
	client, err := cache.GetLocalClient(context.Background())
	if err != nil {
		return fmt.Errorf("failed to connect to local cache: %w", err)
	}
	defer func() { _ = client.Close() }()

	// 1. Get all services to use as prefixes
	allServices := config.GetAllServices()

	// 2. Scan all keys in the database
	ui.PrintInfo("Scanning all keys (this may take a moment)...")
	allKeys, err := scanAllKeys(client, "*")
	if err != nil {
		return err
	}
	ui.PrintInfo(fmt.Sprintf("Found %d total keys.", len(allKeys)))

	// 3. Group keys by service
	serviceMap := make(map[string][]string)
	otherKeys := []string{}
	for _, key := range allKeys {
		found := false
		for _, service := range allServices {
			prefix := service.ID + ":"
			if strings.HasPrefix(key, prefix) {
				serviceMap[service.ID] = append(serviceMap[service.ID], key)
				found = true
				break
			}
		}
		if !found {
			otherKeys = append(otherKeys, key)
		}
	}

	// 4. Create table and get memory usage
	table := ui.NewTable([]string{"SERVICE", "KEY COUNT", "TOTAL SIZE"})
	ctx := context.Background()

	// Process services, sorted by ID
	serviceIDs := make([]string, 0, len(serviceMap))
	for id := range serviceMap {
		serviceIDs = append(serviceIDs, id)
	}
	sort.Strings(serviceIDs)

	for _, serviceID := range serviceIDs {
		keys := serviceMap[serviceID]
		totalBytes, err := getMemoryForKeys(client, ctx, keys)
		if err != nil {
			return err
		}
		// Resolve ID to ShortName for display
		def, err := config.ResolveByID(serviceID)
		if err != nil {
			continue // Should not happen
		}
		table.AddRow([]string{
			def.ShortName,
			fmt.Sprintf("%d", len(keys)),
			FormatBytes(totalBytes),
		})
	}

	// 5. Add "other" keys
	if len(otherKeys) > 0 {
		totalBytes, err := getMemoryForKeys(client, ctx, otherKeys)
		if err != nil {
			return err
		}
		table.AddRow([]string{
			ui.Colorize("(other)", ui.ColorDarkGray),
			fmt.Sprintf("%d", len(otherKeys)),
			FormatBytes(totalBytes),
		})
	}

	if len(allKeys) == 0 {
		ui.PrintInfo("Cache is empty.")
		return nil
	}

	table.Render()
	return nil
}

// scanAllKeys uses SCAN to safely iterate over all keys.
func scanAllKeys(client *redis.Client, match string) ([]string, error) {
	var cursor uint64
	var keys []string
	ctx := context.Background()

	for {
		var scanKeys []string
		var err error
		scanKeys, cursor, err = client.Scan(ctx, cursor, match, 1000).Result()
		if err != nil {
			return nil, fmt.Errorf("redis scan failed: %w", err)
		}
		keys = append(keys, scanKeys...)
		if cursor == 0 {
			break
		}
	}
	return keys, nil
}

// getMemoryForKeys iterates a list of keys and sums their MEMORY USAGE.
func getMemoryForKeys(client *redis.Client, ctx context.Context, keys []string) (int64, error) {
	var totalBytes int64
	// We can't use a simple pipeline, as MEMORY USAGE might fail (e.g., key expired)
	for _, key := range keys {
		bytes, err := client.MemoryUsage(ctx, key).Result()
		if err != nil {
			// Key might have expired between SCAN and MEMORY USAGE, just skip it
			if err == redis.Nil {
				continue
			}
			ui.PrintWarning(fmt.Sprintf("Could not get memory for key %s: %v", key, err))
			continue
		}
		totalBytes += bytes
	}
	return totalBytes, nil
}
