package provider

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/ankek/terraform-provider-cartography/internal/parser"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// LoadResources loads Terraform resources from various sources with automatic backend detection
func LoadResources(ctx context.Context, providerConfig *CartographyProviderModel, statePath, configPath types.String) ([]parser.Resource, error) {
	// Priority 1: If state_path is explicitly provided, use it
	if !statePath.IsNull() && statePath.ValueString() != "" {
		return parser.ParseStateFile(statePath.ValueString())
	}

	// Priority 2: If config_path is provided, try backend detection then HCL parsing
	if !configPath.IsNull() && configPath.ValueString() != "" {
		configDir := configPath.ValueString()

		// Try to parse backend configuration
		backend, err := parser.ParseBackendConfig(configDir)
		if err != nil {
			// If backend parsing fails, fall back to HCL parsing
			return parser.ParseConfigDirectory(configDir)
		}

		// Try to load from backend
		resources, err := loadFromBackend(ctx, providerConfig, backend)
		if err != nil {
			// If backend loading fails, fall back to HCL parsing
			return parser.ParseConfigDirectory(configDir)
		}

		return resources, nil
	}

	// Priority 3: Auto-detect in current directory
	workingDir := "."

	// Try backend detection in current directory
	backend, err := parser.ParseBackendConfig(workingDir)
	if err == nil {
		resources, err := loadFromBackend(ctx, providerConfig, backend)
		if err == nil {
			return resources, nil
		}
	}

	// Try auto-detect state file
	detectedStatePath, err := parser.AutoDetectStatePath(workingDir)
	if err == nil {
		return parser.ParseStateFile(detectedStatePath)
	}

	// Last resort: parse HCL files in current directory
	resources, err := parser.ParseConfigDirectory(workingDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load resources: no state file found and HCL parsing failed: %w", err)
	}

	return resources, nil
}

// loadFromBackend loads resources from a backend configuration
func loadFromBackend(ctx context.Context, providerConfig *CartographyProviderModel, backend *parser.BackendConfig) ([]parser.Resource, error) {
	// For local backend, use file-based loading
	if parser.BackendType(backend.Type) == parser.BackendTypeLocal {
		statePath, err := parser.GetStatePath(backend)
		if err != nil {
			return nil, err
		}
		return parser.ParseStateFile(statePath)
	}

	// For remote backends, fetch state and parse
	remoteConfig := &parser.RemoteStateConfig{
		Backend: backend,
	}

	// Set credentials from provider configuration or environment
	if providerConfig != nil {
		if !providerConfig.TerraformToken.IsNull() {
			remoteConfig.TerraformToken = providerConfig.TerraformToken.ValueString()
		}
		if !providerConfig.AWSAccessKey.IsNull() {
			remoteConfig.AWSAccessKey = providerConfig.AWSAccessKey.ValueString()
		}
		if !providerConfig.AWSSecretKey.IsNull() {
			remoteConfig.AWSSecretKey = providerConfig.AWSSecretKey.ValueString()
		}
		if !providerConfig.AzureAccount.IsNull() {
			remoteConfig.AzureAccount = providerConfig.AzureAccount.ValueString()
		}
		if !providerConfig.AzureKey.IsNull() {
			remoteConfig.AzureKey = providerConfig.AzureKey.ValueString()
		}
		if !providerConfig.GCPCredentials.IsNull() {
			remoteConfig.GCPCredentials = providerConfig.GCPCredentials.ValueString()
		}
	}

	return parser.LoadStateFromBackend(ctx, remoteConfig)
}

// ResolveWorkingDirectory resolves the working directory from state_path or config_path
func ResolveWorkingDirectory(statePath, configPath types.String) string {
	if !statePath.IsNull() && statePath.ValueString() != "" {
		return filepath.Dir(statePath.ValueString())
	}
	if !configPath.IsNull() && configPath.ValueString() != "" {
		return configPath.ValueString()
	}
	return "."
}
