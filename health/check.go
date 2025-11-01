package health

// StatusResponse represents the expected JSON response from a service's /status endpoint
type StatusResponse struct {
	Service   string `json:"service"`
	Version   string `json:"version"`
	Status    string `json:"status"`
	Uptime    int    `json:"uptime"`
	Timestamp int64  `json:"timestamp"`
}
