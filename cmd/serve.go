package cmd

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/EasterCompany/dex-cli/ui"
)

// Serve starts a static file server for frontend projects.
func Serve(args []string, version, branch, commit, buildDate string) error {
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

		// Check for version.txt in the served directory
		currentVersion := version
		versionFile := filepath.Join(absDir, "version.txt")
		if data, err := os.ReadFile(versionFile); err == nil {
			contentVersion := strings.TrimSpace(string(data))
			if contentVersion != "" {
				currentVersion = contentVersion
			}
		}

		// Local struct to match what 'dex status' expects
		type serviceHealth struct {
			Status string `json:"status"`
			Uptime string `json:"uptime"`
		}

		type metricValue struct {
			Avg float64 `json:"avg"`
		}

		type serviceMetrics struct {
			CPU    metricValue `json:"cpu"`
			Memory metricValue `json:"memory"`
		}

		type versionObj struct {
			Branch    string `json:"branch"`
			Commit    string `json:"commit"`
			BuildDate string `json:"build_date"`
		}

		type serviceVersion struct {
			Str string     `json:"str"`
			Obj versionObj `json:"obj"`
		}

		type serviceReport struct {
			Version serviceVersion `json:"version"`
			Health  serviceHealth  `json:"health"`
			Metrics serviceMetrics `json:"metrics"`
		}

		// Collect simple metrics
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		memUsage := float64(m.Alloc) / 1024 / 1024 // MB
		// Simple CPU proxy: NumGoroutine
		cpuUsage := float64(runtime.NumGoroutine())

		// Construct report
		report := serviceReport{
			Version: serviceVersion{
				Str: currentVersion,
				Obj: versionObj{
					Branch:    branch,
					Commit:    commit,
					BuildDate: buildDate,
				},
			},
			Health: serviceHealth{
				Status: "OK",
				Uptime: time.Since(startTime).String(),
			},
			Metrics: serviceMetrics{
				CPU:    metricValue{Avg: cpuUsage},
				Memory: metricValue{Avg: memUsage},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(report); err != nil {
			log.Printf("ERROR: Failed to encode service report: %v", err)
		}
	})

	// Serve static files for all other routes with custom 404 support
	fileServer := http.FileServer(http.Dir(absDir))
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if file exists using the http.FileSystem interface
		// This ensures we follow the same path resolution rules as the FileServer
		f, err := http.Dir(absDir).Open(r.URL.Path)
		if err != nil {
			// File likely doesn't exist or other error
			serve404(w, r, absDir)
			return
		}
		defer func() { _ = f.Close() }()

		// If it is a directory, ensure index.html exists, otherwise 404
		info, err := f.Stat()
		if err == nil && info.IsDir() {
			indexFile, err := http.Dir(absDir).Open(filepath.Join(r.URL.Path, "index.html"))
			if err != nil {
				// No index.html in directory -> treat as 404 (disable dir listing)
				serve404(w, r, absDir)
				return
			}
			_ = indexFile.Close()
		}

		fileServer.ServeHTTP(w, r)
	})

	mux.Handle("/", loggingMiddleware(handler))

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

func serve404(w http.ResponseWriter, r *http.Request, root string) {
	fourOhFourPath := filepath.Join(root, "404.html")
	if _, err := os.Stat(fourOhFourPath); err == nil {
		w.WriteHeader(http.StatusNotFound)
		http.ServeFile(w, r, fourOhFourPath)
	} else {
		http.NotFound(w, r)
	}
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
