package cmd

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/EasterCompany/dex-cli/cache"
	"github.com/EasterCompany/dex-cli/ui"
	"github.com/EasterCompany/dex-cli/utils"
)

// Cache manages the local cache.
func Cache(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("cache command requires a subcommand (clear, list)")
	}

	subcommand := args[0]
	switch subcommand {
	case "clear":
		return clearCache()
	case "list":
		return listCache()
	default:
		return fmt.Errorf("unknown cache subcommand: %s", subcommand)
	}
}

func clearCache() error {
	ctx := context.Background()
	client, err := cache.GetLocalClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to get local cache client: %w", err)
	}
	defer func() { _ = client.Close() }()

	if err := client.FlushDB(ctx).Err(); err != nil {
		return fmt.Errorf("failed to clear cache: %w", err)
	}

	ui.PrintSuccess("Local cache cleared.")
	return nil
}

type CacheEntry struct {
	Key    string
	Size   int64
	Expiry time.Time
}

func listCache() error {
	ctx := context.Background()
	client, err := cache.GetLocalClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to get local cache client: %w", err)
	}
	defer func() { _ = client.Close() }()

	keys, err := client.Keys(ctx, "*").Result()
	if err != nil {
		return fmt.Errorf("failed to list cache keys: %w", err)
	}

	if len(keys) == 0 {
		ui.PrintInfo("Local cache is empty.")
		return nil
	}

	var entries []CacheEntry
	var totalSize int64

	for _, key := range keys {
		size, err := client.MemoryUsage(ctx, key).Result()
		if err != nil {
			size = 0 // Default to 0 if size can't be fetched
		}
		totalSize += size

		ttl, err := client.TTL(ctx, key).Result()
		if err != nil {
			ttl = -1 // No expiry
		}
		expiry := time.Now().Add(ttl)
		if ttl == -1 {
			expiry = time.Time{} // Zero time for no expiry
		}

		entries = append(entries, CacheEntry{
			Key:    key,
			Size:   size,
			Expiry: expiry,
		})
	}

	// Sort entries by key
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Key < entries[j].Key
	})

	ui.PrintHeader("Local Cache Entries")
	for _, entry := range entries {
		ui.PrintInfo(fmt.Sprintf("Key: %s", entry.Key))
		ui.PrintInfo(fmt.Sprintf("  Size: %s", utils.FormatBytes(entry.Size)))
		if !entry.Expiry.IsZero() {
			ui.PrintInfo(fmt.Sprintf("  Expires: %s", entry.Expiry.Format(time.RFC3339)))
		} else {
			ui.PrintInfo("  Expires: (no expiry)")
		}
	}
	ui.PrintInfo(fmt.Sprintf("Total cache size: %s", utils.FormatBytes(totalSize)))

	return nil
}
