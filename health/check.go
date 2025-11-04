package health

// ServiceReport is the unified response from the /service endpoint.
type ServiceReport struct {
	Version struct {
		Tag string `json:"tag"`
		Dev bool   `json:"dev"`
		Str string `json:"str"`
		Obj struct {
			Major      string `json:"major"`
			Minor      string `json:"minor"`
			Patch      string `json:"patch"`
			PreRelease string `json:"pre_release"`
			Branch     string `json:"branch"`
			Commit     string `json:"commit"`
			BuildDate  string `json:"build_date"`
			Arch       string `json:"arch"`
			Random     string `json:"random"`
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
