package health

// StatusResponse represents the expected JSON response from a service's /status endpoint
type StatusResponse struct {
	Service   string  `json:"service"`
	Version   string  `json:"version"`
	Status    string  `json:"status"`
	Uptime    int     `json:"uptime"`
	Timestamp int64   `json:"timestamp"`
	Metrics   Metrics `json:"metrics"`
}

// Metrics holds detailed performance metrics from a service
type Metrics struct {
	Goroutines      int     `json:"goroutines"`
	MemoryAllocMB   float64 `json:"memory_alloc_mb"`
	EventsReceived  uint64  `json:"events_received,omitempty"`
	EventsProcessed uint64  `json:"events_processed,omitempty"`
}
