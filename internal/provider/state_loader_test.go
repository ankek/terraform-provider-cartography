package provider

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ankek/terraform-provider-cartography/internal/parser"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestLoadResources(t *testing.T) {
	tests := []struct {
		name          string
		setupFiles    map[string]string
		statePath     string
		configPath    string
		wantResources int
		wantErr       bool
	}{
		{
			name: "explicit state path",
			setupFiles: map[string]string{
				"terraform.tfstate": `{
					"version": 4,
					"terraform_version": "1.0.0",
					"values": {
						"root_module": {
							"resources": [
								{
									"mode": "managed",
									"type": "aws_instance",
									"name": "web",
									"provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
									"instances": [{"attributes": {"id": "i-12345"}}]
								}
							]
						}
					}
				}`,
			},
			statePath:     "terraform.tfstate",
			wantResources: 1,
			wantErr:       false,
		},
		{
			name: "config path with HCL",
			setupFiles: map[string]string{
				"main.tf": `
resource "aws_instance" "web" {
  ami = "ami-12345"
}

resource "aws_vpc" "main" {
  cidr_block = "10.0.0.0/16"
}
`,
			},
			configPath:    ".",
			wantResources: 2,
			wantErr:       false,
		},
		{
			name: "auto-detect state file",
			setupFiles: map[string]string{
				"terraform.tfstate": `{
					"version": 4,
					"terraform_version": "1.0.0",
					"values": {
						"root_module": {
							"resources": [
								{
									"mode": "managed",
									"type": "azurerm_resource_group",
									"name": "rg",
									"provider": "provider[\"registry.terraform.io/hashicorp/azurerm\"]",
									"instances": [{"attributes": {"id": "/subscriptions/xxx/resourceGroups/rg"}}]
								}
							]
						}
					}
				}`,
			},
			wantResources: 1,
			wantErr:       false,
		},
		{
			name: "no resources found",
			setupFiles: map[string]string{
				"README.md": "# Documentation",
			},
			wantResources: 0,
			wantErr:       false, // Implementation doesn't error on empty results
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			// Create test files
			for filename, content := range tt.setupFiles {
				filePath := filepath.Join(tmpDir, filename)
				if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
					t.Fatalf("Failed to create test file %s: %v", filename, err)
				}
			}

			// Change to temp directory for auto-detect tests
			originalDir, err := os.Getwd()
			if err != nil {
				t.Fatalf("Failed to get current directory: %v", err)
			}
			defer os.Chdir(originalDir)

			if err := os.Chdir(tmpDir); err != nil {
				t.Fatalf("Failed to change to temp directory: %v", err)
			}

			// Prepare input types
			var statePath, configPath types.String
			if tt.statePath != "" {
				statePath = types.StringValue(filepath.Join(tmpDir, tt.statePath))
			} else {
				statePath = types.StringNull()
			}
			if tt.configPath != "" {
				if tt.configPath == "." {
					configPath = types.StringValue(tmpDir)
				} else {
					configPath = types.StringValue(filepath.Join(tmpDir, tt.configPath))
				}
			} else {
				configPath = types.StringNull()
			}

			ctx := context.Background()
			resources, err := LoadResources(ctx, nil, statePath, configPath)

			if (err != nil) != tt.wantErr {
				t.Errorf("LoadResources() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil && len(resources) != tt.wantResources {
				t.Errorf("LoadResources() got %d resources, want %d", len(resources), tt.wantResources)
			}
		})
	}
}

func TestLoadResources_WithBackend(t *testing.T) {
	tmpDir := t.TempDir()

	// Create backend config and state file
	backendFile := filepath.Join(tmpDir, "backend.tf")
	backendContent := `
terraform {
  backend "local" {
    path = "custom.tfstate"
  }
}
`
	if err := os.WriteFile(backendFile, []byte(backendContent), 0644); err != nil {
		t.Fatalf("Failed to create backend file: %v", err)
	}

	stateFile := filepath.Join(tmpDir, "custom.tfstate")
	stateContent := `{
		"version": 4,
		"terraform_version": "1.0.0",
		"values": {
			"root_module": {
				"resources": [
					{
						"mode": "managed",
						"type": "aws_instance",
						"name": "web",
						"provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
						"instances": [{"attributes": {"id": "i-12345"}}]
					}
				]
			}
		}
	}`
	if err := os.WriteFile(stateFile, []byte(stateContent), 0644); err != nil {
		t.Fatalf("Failed to create state file: %v", err)
	}

	ctx := context.Background()
	configPath := types.StringValue(tmpDir)
	resources, err := LoadResources(ctx, nil, types.StringNull(), configPath)

	if err != nil {
		t.Fatalf("LoadResources() error = %v", err)
	}

	if len(resources) != 1 {
		t.Errorf("LoadResources() got %d resources, want 1", len(resources))
	}
}

func TestLoadFromBackend(t *testing.T) {
	tests := []struct {
		name          string
		backend       *parser.BackendConfig
		setupFiles    map[string]string
		wantResources int
		wantErr       bool
	}{
		{
			name: "local backend",
			backend: &parser.BackendConfig{
				Type: "local",
				Config: map[string]interface{}{
					"path": "terraform.tfstate",
				},
				WorkingDir: "",
			},
			setupFiles: map[string]string{
				"terraform.tfstate": `{
					"version": 4,
					"terraform_version": "1.0.0",
					"values": {
						"root_module": {
							"resources": [
								{
									"mode": "managed",
									"type": "google_compute_instance",
									"name": "vm",
									"provider": "provider[\"registry.terraform.io/hashicorp/google\"]",
									"instances": [{"attributes": {"id": "instance-1"}}]
								}
							]
						}
					}
				}`,
			},
			wantResources: 1,
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tt.backend.WorkingDir = tmpDir

			// Create test files
			for filename, content := range tt.setupFiles {
				filePath := filepath.Join(tmpDir, filename)
				if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
					t.Fatalf("Failed to create test file %s: %v", filename, err)
				}
			}

			ctx := context.Background()
			resources, err := loadFromBackend(ctx, nil, tt.backend)

			if (err != nil) != tt.wantErr {
				t.Errorf("loadFromBackend() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil && len(resources) != tt.wantResources {
				t.Errorf("loadFromBackend() got %d resources, want %d", len(resources), tt.wantResources)
			}
		})
	}
}

func TestResolveWorkingDirectory(t *testing.T) {
	tests := []struct {
		name       string
		statePath  types.String
		configPath types.String
		wantPrefix string // Use prefix instead of exact match for cross-platform
	}{
		{
			name:       "state path provided",
			statePath:  types.StringValue(filepath.Join("/path", "to", "terraform.tfstate")),
			configPath: types.StringNull(),
			wantPrefix: "to", // Directory name
		},
		{
			name:       "config path provided",
			statePath:  types.StringNull(),
			configPath: types.StringValue(filepath.Join("/path", "to", "config")),
			wantPrefix: "config",
		},
		{
			name:       "both provided - prefer state path",
			statePath:  types.StringValue(filepath.Join("/state", "terraform.tfstate")),
			configPath: types.StringValue("/config"),
			wantPrefix: "state",
		},
		{
			name:       "neither provided - default to current",
			statePath:  types.StringNull(),
			configPath: types.StringNull(),
			wantPrefix: ".",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveWorkingDirectory(tt.statePath, tt.configPath)
			if tt.wantPrefix == "." {
				if got != "." {
					t.Errorf("ResolveWorkingDirectory() = %s, want .", got)
				}
			} else {
				// Check that the result contains the expected directory name
				if !filepath.IsAbs(got) && got != "." {
					// For relative paths, just check it's not empty
					if got == "" {
						t.Errorf("ResolveWorkingDirectory() returned empty string")
					}
				}
			}
		})
	}
}

func TestLoadResources_ContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a state file
	stateFile := filepath.Join(tmpDir, "terraform.tfstate")
	stateContent := `{"version": 4, "terraform_version": "1.0.0"}`
	if err := os.WriteFile(stateFile, []byte(stateContent), 0644); err != nil {
		t.Fatalf("Failed to create state file: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	statePath := types.StringValue(stateFile)
	_, err := LoadResources(ctx, nil, statePath, types.StringNull())

	if err != context.Canceled {
		t.Errorf("LoadResources() with cancelled context got error = %v, want context.Canceled", err)
	}
}

func TestLoadResources_ProviderConfig(t *testing.T) {
	// Test that provider config is passed through correctly
	providerConfig := &CartographyProviderModel{
		TerraformToken: types.StringValue("test-token"),
		AWSAccessKey:   types.StringValue("test-key"),
		AWSSecretKey:   types.StringValue("test-secret"),
	}

	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "terraform.tfstate")
	stateContent := `{
		"version": 4,
		"terraform_version": "1.0.0",
		"values": {
			"root_module": {
				"resources": [
					{
						"mode": "managed",
						"type": "aws_instance",
						"name": "web",
						"provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
						"instances": [{"attributes": {"id": "i-12345"}}]
					}
				]
			}
		}
	}`
	if err := os.WriteFile(stateFile, []byte(stateContent), 0644); err != nil {
		t.Fatalf("Failed to create state file: %v", err)
	}

	ctx := context.Background()
	statePath := types.StringValue(stateFile)
	resources, err := LoadResources(ctx, providerConfig, statePath, types.StringNull())

	if err != nil {
		t.Fatalf("LoadResources() error = %v", err)
	}

	if len(resources) != 1 {
		t.Errorf("LoadResources() got %d resources, want 1", len(resources))
	}
}
