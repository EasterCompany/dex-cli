package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/ui"
)

// Add prompts the user to create a new service in the new service map.
func Add(args []string) error {
	reader := bufio.NewReader(os.Stdin)

	var serviceName string
	var allowedService config.AllowedService
	for {
		fmt.Print("Enter service name (allowed: event, model, chat, tts, stt, discord): ")
		serviceName, _ = reader.ReadString('\n')
		serviceName = strings.TrimSpace(serviceName)

		if s, ok := config.AllowedServices[serviceName]; ok {
			allowedService = s
			break
		}
		fmt.Println("Invalid service name. Please choose from the allowed options.")
	}

	serviceType := allowedService.Type
	port := allowedService.Port
	address := "0.0.0.0"

	serviceID := fmt.Sprintf("dex-%s-service", serviceName)
	repo := fmt.Sprintf("git@github.com:EasterCompany/%s", serviceID)
	source := fmt.Sprintf("~/EasterCompany/%s", serviceID)

	service := config.ServiceEntry{
		ID:     serviceID,
		Repo:   repo,
		Source: source,
	}

	if serviceType != "cli" {
		service.HTTP = fmt.Sprintf("http://%s:%s", address, port)
		service.Socket = fmt.Sprintf("ws://%s:%s", address, port)
	}

	if serviceType == "os" {
		creds := &config.ServiceCredentials{}
		fmt.Print("Enter username: ")
		username, _ := reader.ReadString('\n')
		creds.Username = strings.TrimSpace(username)

		fmt.Print("Enter password: ")
		password, _ := reader.ReadString('\n')
		creds.Password = strings.TrimSpace(password)

		service.Credentials = creds
	}

	// Load existing service map
	serviceMap, err := config.LoadServiceMapConfig()
	if err != nil {
		// If the file doesn't exist, create a new map
		if os.IsNotExist(err) {
			serviceMap = config.DefaultServiceMapConfig()
		} else {
			return fmt.Errorf("failed to load service map: %w", err)
		}
	}

	// Add the new service
	serviceMap.Services[serviceType] = append(serviceMap.Services[serviceType], service)

	// Save the updated service map
	if err := config.SaveServiceMapConfig(serviceMap); err != nil {
		return fmt.Errorf("failed to save service map: %w", err)
	}

	ui.PrintInfo(fmt.Sprintf("Service '%s' added successfully.", serviceID))
	return nil
}
