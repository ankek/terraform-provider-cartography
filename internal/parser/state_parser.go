package parser

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// TerraformState represents the structure of a terraform.tfstate file
type TerraformState struct {
	Version          int                `json:"version"`
	TerraformVersion string             `json:"terraform_version"`
	Resources        []StateResource    `json:"resources"`        // Legacy format (v3 and below)
	Values           *StateValues       `json:"values,omitempty"` // Modern format (v4+)
}

// StateValues represents the values section in modern state files
type StateValues struct {
	RootModule *StateModule `json:"root_module,omitempty"`
}

// StateModule represents a module in the state file
type StateModule struct {
	Resources []StateResource `json:"resources,omitempty"`
}

// StateResource represents a resource in the state file
type StateResource struct {
	Mode      string                   `json:"mode"`
	Type      string                   `json:"type"`
	Name      string                   `json:"name"`
	Provider  string                   `json:"provider"`
	Instances []StateResourceInstance  `json:"instances"`
}

// StateResourceInstance represents an instance of a resource
type StateResourceInstance struct {
	Attributes   map[string]interface{} `json:"attributes"`
	Dependencies []string               `json:"dependencies,omitempty"`
}

// ParseStateFile reads and parses a Terraform state file.
// It respects the provided context for cancellation.
func ParseStateFile(ctx context.Context, path string) ([]Resource, error) {
	// Check if context is already cancelled
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state TerraformState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state file: %w", err)
	}

	// Determine which format we're dealing with
	var stateResources []StateResource
	if state.Values != nil && state.Values.RootModule != nil {
		// Modern format (v4+): use values.root_module.resources
		stateResources = state.Values.RootModule.Resources
	} else {
		// Legacy format (v3 and below): use resources at root level
		stateResources = state.Resources
	}

	var resources []Resource
	for _, stateRes := range stateResources {
		// Skip data sources, only process managed resources
		if stateRes.Mode != "managed" {
			continue
		}

		provider := extractProvider(stateRes.Type)

		for idx, instance := range stateRes.Instances {
			// Generate ID - use simple format for single instances, indexed for multiple
			var resourceID string
			if len(stateRes.Instances) == 1 {
				// Single instance: use simple ID format that matches dependency references
				resourceID = fmt.Sprintf("%s.%s", stateRes.Type, stateRes.Name)
			} else {
				// Multiple instances: include index
				resourceID = fmt.Sprintf("%s.%s[%d]", stateRes.Type, stateRes.Name, idx)
			}

			resource := Resource{
				Type:         stateRes.Type,
				Name:         stateRes.Name,
				Provider:     provider,
				Attributes:   instance.Attributes,
				ID:           resourceID,
				Dependencies: instance.Dependencies,
			}

			resources = append(resources, resource)
		}
	}

	return resources, nil
}

// extractProvider determines the cloud provider from the resource type
func extractProvider(resourceType string) string {
	if strings.HasPrefix(resourceType, "azurerm_") {
		return "azure"
	} else if strings.HasPrefix(resourceType, "aws_") {
		return "aws"
	} else if strings.HasPrefix(resourceType, "google_") {
		return "gcp"
	} else if strings.HasPrefix(resourceType, "digitalocean_") {
		return "digitalocean"
	}
	return "unknown"
}
