package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/EasterCompany/dex-cli/cache"
	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/health"
)

const cacheTTL = 30 * time.Second

// GetServiceReport fetches a service's status, using a cache to avoid repeated requests.
func GetServiceReport(service config.ServiceDefinition) (*health.ServiceReport, error) {
	// 1. Check cache first
	cacheKey := fmt.Sprintf("service-report:%s", service.ID)
	if cachedReport, err := getReportFromCache(cacheKey); err == nil {
		return cachedReport, nil // Cache hit
	}

	// 2. If cache miss, fetch from the /service endpoint
	freshReport, err := fetchReportFromEndpoint(service)
	if err != nil {
		return nil, err
	}

	// 3. Save the fresh report to the cache
	if err := setReportToCache(cacheKey, freshReport); err != nil {
		// Log the caching error but don't fail the whole operation
		// In a real CLI, you'd have a proper logger.
		fmt.Printf("Warning: Failed to cache service report for %s: %v\n", service.ID, err)
	}

	return freshReport, nil
}

func getReportFromCache(key string) (*health.ServiceReport, error) {
	ctx := context.Background()
	client, err := cache.GetLocalClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get cache client: %w", err)
	}

	val, err := client.Get(ctx, key).Result()
	if err != nil {
		return nil, err // Includes redis.Nil for cache miss
	}

	var report health.ServiceReport
	if err := json.Unmarshal([]byte(val), &report); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cached report: %w", err)
	}

	return &report, nil
}

func fetchReportFromEndpoint(service config.ServiceDefinition) (*health.ServiceReport, error) {
	serviceURL := service.GetHTTP("/service")

	client := http.Client{
		Timeout: 2 * time.Second,
	}

	resp, err := client.Get(serviceURL)
	if err != nil {
		return nil, fmt.Errorf("failed to GET /service endpoint: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("service endpoint returned non-OK status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var report health.ServiceReport
	if err := json.Unmarshal(body, &report); err != nil {
		return nil, fmt.Errorf("failed to unmarshal service report: %w", err)
	}

	return &report, nil
}

func setReportToCache(key string, report *health.ServiceReport) error {
	ctx := context.Background()
	client, err := cache.GetLocalClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to get cache client: %w", err)
	}

	data, err := json.Marshal(report)
	if err != nil {
		return fmt.Errorf("failed to marshal report for caching: %w", err)
	}

	return client.Set(ctx, key, data, cacheTTL).Err()
}
