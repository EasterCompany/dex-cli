package cmd

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/EasterCompany/dex-cli/ui"
)

// Serve starts a static file server for frontend projects.
func Serve(args []string) error {
	var dir string
	var port int

	serveFlagSet := flag.NewFlagSet("serve", flag.ExitOnError)
	serveFlagSet.StringVar(&dir, "dir", "", "Directory to serve static files from")
	serveFlagSet.StringVar(&dir, "d", "", "Directory to serve static files from (shorthand)")
	serveFlagSet.IntVar(&port, "port", 0, "Port to listen on")
	serveFlagSet.IntVar(&port, "p", 0, "Port to listen on (shorthand)")

	if err := serveFlagSet.Parse(args); err != nil {
		return fmt.Errorf("failed to parse flags for serve command: %w", err)
	}

	if dir == "" {
		return fmt.Errorf("ERROR: --dir flag is required for serve command")
	}
	if port == 0 {
		return fmt.Errorf("ERROR: --port flag is required for serve command")
	}

	// Resolve absolute path for the directory
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("ERROR: Invalid directory path '%s': %w", dir, err)
	}

	// Check if directory exists
	if _, err := os.Stat(absDir); os.IsNotExist(err) {
		return fmt.Errorf("ERROR: Directory '%s' does not exist", absDir)
	}

	addr := fmt.Sprintf(":%d", port)
	ui.PrintInfo(fmt.Sprintf("Starting static file server for directory '%s' on %s", absDir, addr))

	startTime := time.Now()

	// Create a custom ServeMux to add logging and health check
	mux := http.NewServeMux()

	// Register standard service health endpoint
	mux.HandleFunc("/service", func(w http.ResponseWriter, r *http.Request) {
		// Log the request
		log.Printf("ACCESS: %s %s %s %s", r.RemoteAddr, r.Method, r.URL.Path, time.Since(startTime))

		// Local struct to match what 'dex status' expects (specifically Uptime as string)
		type serviceHealth struct {
			Timestamp int64  `json:"timestamp"`
			Uptime    string `json:"uptime"`
		}

		type serviceReport struct {
			Version struct {
				Str string `json:"str"`
			} `json:"version"`
			Status string        `json:"status"`
			Health serviceHealth `json:"health"`
		}

		// Construct report
		report := serviceReport{
			Status: "OK",
			Health: serviceHealth{
				Timestamp: time.Now().Unix(),
				Uptime:    time.Since(startTime).String(),
			},
		}
		report.Version.Str = RunningVersion

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(report); err != nil {
			log.Printf("ERROR: Failed to encode service report: %v", err)
		}
	})

	// Serve static files for all other routes
	mux.Handle("/", loggingMiddleware(http.FileServer(http.Dir(absDir))))

	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	ui.PrintInfo(fmt.Sprintf("Server listening on %s", addr))
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("ERROR: Server failed to start: %w", err)
	}
	return nil
}

// loggingMiddleware logs every HTTP request.
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		duration := time.Since(start)
		log.Printf("ACCESS: %s %s %s %s", r.RemoteAddr, r.Method, r.URL.Path, duration)
	})
}
