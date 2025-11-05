package health

// ServiceReport is the unified response from the /service endpoint.
type ServiceReport struct {
	Version struct {
		Tag string `json:"tag"`
		Str string `json:"str"`
		Obj struct {
			Major     string `json:"major"`
			Minor     string `json:"minor"`
			Patch     string `json:"patch"`
			Branch    string `json:"branch"`
			Commit    string `json:"commit"`
			BuildDate string `json:"build_date"`
			Arch      string `json:"arch"`
			BuildHash string `json:"build_hash"`
		} `json:"obj"`
	} `json:"version"`
	Status string   `json:"status"`
	Health Health   `json:"health"`
	Logs   []string `json:"logs"`
}

// Health holds detailed performance metrics from a service
type Health struct {
	Timestamp int64   `json:"timestamp"`
	Uptime    float64 `json:"uptime"`
	Metrics   Metrics `json:"metrics"`
}

// Metrics holds detailed performance metrics from a service
type Metrics struct {
	Goroutines      int     `json:"goroutines"`
	MemoryAllocMB   float64 `json:"memory_alloc_mb"`
	EventsReceived  uint64  `json:"events_received,omitempty"`
	EventsProcessed uint64  `json:"events_processed,omitempty"`
}
