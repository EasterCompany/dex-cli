package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/EasterCompany/dex-cli/config"
)

// Build compiles one or all services
func Build(args []string) error {
	logFile, err := config.LogFile()
	if err != nil {
		return fmt.Errorf("failed to get log file: %w", err)
	}
	defer func() { _ = logFile.Close() }()

	log := func(message string) {
		_, _ = fmt.Fprintln(logFile, message)
	}

	log(fmt.Sprintf("Build command called with args: %v", args))

	// Load the service map
	serviceMap, err := config.LoadServiceMap()
	if err != nil {
		return fmt.Errorf("failed to load service map: %w", err)
	}

	// Determine which services to build
	servicesToBuild := []config.ServiceEntry{}
	if len(args) == 0 || (len(args) > 0 && args[0] == "all") {
		for _, services := range serviceMap.Services {
			for _, service := range services {
				if strings.HasPrefix(service.ID, "dex-") && service.ID != "dex-cli" {
					servicesToBuild = append(servicesToBuild, service)
				}
			}
		}
	} else {
		for _, arg := range args {
			serviceName := arg
			if !strings.HasPrefix(serviceName, "dex-") {
				serviceName = "dex-" + serviceName + "-service"
			}
			found := false
			for _, services := range serviceMap.Services {
				for _, service := range services {
					if service.ID == serviceName {
						servicesToBuild = append(servicesToBuild, service)
						found = true
						break
					}
				}
				if found {
					break
				}
			}
			if !found {
				return fmt.Errorf("service '%s' not found", arg)
			}
		}
	}

	// Build logic
	for _, service := range servicesToBuild {
		if err := buildService(service, log); err != nil {
			fmt.Printf("Failed to build %s: %v\n", service.ID, err)
			log(fmt.Sprintf("Failed to build %s: %v", service.ID, err))
		}
	}

	fmt.Println("All services built")
	log("All services built.")
	return nil
}

func buildService(service config.ServiceEntry, log func(string)) error {
	if service.Source == "" || service.Source == "system" {
		fmt.Printf("Skipping %s: no source path defined\n", service.ID)
		log(fmt.Sprintf("Skipping %s: no source path defined", service.ID))
		return nil
	}

	sourcePath, err := config.ExpandPath(service.Source)
	if err != nil {
		return fmt.Errorf("failed to expand source path for %s: %w", service.ID, err)
	}

	if _, err := os.Stat(filepath.Join(sourcePath, "go.mod")); os.IsNotExist(err) {
		fmt.Printf("Skipping %s: not a Go project (no go.mod)\n", service.ID)
		log(fmt.Sprintf("Skipping %s: not a Go project (no go.mod)", service.ID))
		return nil
	}

	fmt.Printf("Building %s...\n", service.ID)
	log(fmt.Sprintf("Building %s...", service.ID))

	dexterBinPath, err := config.ExpandPath("~/Dexter/bin")
	if err != nil {
		return err
	}

	cmd := exec.Command("go", "build", "-o", filepath.Join(dexterBinPath, service.ID))
	cmd.Dir = sourcePath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to build %s: %w", service.ID, err)
	}

	fmt.Printf("%s built successfully\n", service.ID)
	log(fmt.Sprintf("%s built successfully", service.ID))
	return nil
}
