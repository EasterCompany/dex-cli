// config/resolver.go
package config

import (
	"fmt"
	"strings"
)

// Resolve maps a service's short name (alias) to its full ServiceDefinition.
func Resolve(shortName string) (*ServiceDefinition, error) {
	shortName = strings.ToLower(shortName)
	for _, def := range GetAllServices() {
		if def.ShortName == shortName {
			return &def, nil
		}
	}
	return nil, fmt.Errorf("service alias '%s' not found", shortName)
}

// ResolveByID maps a service's full ID (e.g., "dex-event-service") to its ServiceDefinition.
func ResolveByID(id string) (*ServiceDefinition, error) {
	id = strings.ToLower(id)
	for _, def := range GetAllServices() {
		if def.ID == id {
			return &def, nil
		}
	}
	return nil, fmt.Errorf("service ID '%s' not found", id)
}
