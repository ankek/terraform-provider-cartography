package parser

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestParseStateFile(t *testing.T) {
	tests := []struct {
		name          string
		stateContent  string
		wantResources int
		wantProvider  string
		wantErr       bool
	}{
		{
			name: "modern state format v4",
			stateContent: `{
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
								"instances": [
									{
										"attributes": {
											"id": "i-12345",
											"instance_type": "t2.micro"
										},
										"dependencies": ["aws_vpc.main"]
									}
								]
							},
							{
								"mode": "managed",
								"type": "aws_vpc",
								"name": "main",
								"provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
								"instances": [
									{
										"attributes": {
											"id": "vpc-12345",
											"cidr_block": "10.0.0.0/16"
										}
									}
								]
							}
						]
					}
				}
			}`,
			wantResources: 2,
			wantProvider:  "aws",
			wantErr:       false,
		},
		{
			name: "legacy state format v3",
			stateContent: `{
				"version": 3,
				"terraform_version": "0.12.0",
				"resources": [
					{
						"mode": "managed",
						"type": "azurerm_virtual_network",
						"name": "vnet",
						"provider": "provider.azurerm",
						"instances": [
							{
								"attributes": {
									"id": "/subscriptions/xxx/resourceGroups/rg/providers/Microsoft.Network/virtualNetworks/vnet",
									"address_space": ["10.0.0.0/16"]
								}
							}
						]
					}
				]
			}`,
			wantResources: 1,
			wantProvider:  "azure",
			wantErr:       false,
		},
		{
			name: "multiple instances",
			stateContent: `{
				"version": 4,
				"terraform_version": "1.0.0",
				"values": {
					"root_module": {
						"resources": [
							{
								"mode": "managed",
								"type": "digitalocean_droplet",
								"name": "web",
								"provider": "provider[\"registry.terraform.io/digitalocean/digitalocean\"]",
								"instances": [
									{
										"attributes": {
											"id": "123456",
											"name": "web-1"
										}
									},
									{
										"attributes": {
											"id": "123457",
											"name": "web-2"
										}
									}
								]
							}
						]
					}
				}
			}`,
			wantResources: 2,
			wantProvider:  "digitalocean",
			wantErr:       false,
		},
		{
			name: "skip data sources",
			stateContent: `{
				"version": 4,
				"terraform_version": "1.0.0",
				"values": {
					"root_module": {
						"resources": [
							{
								"mode": "data",
								"type": "aws_ami",
								"name": "ubuntu",
								"provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
								"instances": [
									{
										"attributes": {
											"id": "ami-12345"
										}
									}
								]
							},
							{
								"mode": "managed",
								"type": "aws_instance",
								"name": "web",
								"provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
								"instances": [
									{
										"attributes": {
											"id": "i-12345"
										}
									}
								]
							}
						]
					}
				}
			}`,
			wantResources: 1,
			wantProvider:  "aws",
			wantErr:       false,
		},
		{
			name:          "invalid json",
			stateContent:  `{invalid json`,
			wantResources: 0,
			wantErr:       true,
		},
		{
			name:          "empty state",
			stateContent:  `{"version": 4, "terraform_version": "1.0.0"}`,
			wantResources: 0,
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file
			tmpDir := t.TempDir()
			stateFile := filepath.Join(tmpDir, "terraform.tfstate")
			if err := os.WriteFile(stateFile, []byte(tt.stateContent), 0644); err != nil {
				t.Fatalf("Failed to create test state file: %v", err)
			}

			// Parse state file
			ctx := context.Background()
			resources, err := ParseStateFile(ctx, stateFile)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseStateFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				if len(resources) != tt.wantResources {
					t.Errorf("ParseStateFile() got %d resources, want %d", len(resources), tt.wantResources)
				}

				if tt.wantResources > 0 && resources[0].Provider != tt.wantProvider {
					t.Errorf("ParseStateFile() got provider %s, want %s", resources[0].Provider, tt.wantProvider)
				}
			}
		})
	}
}

func TestParseStateFile_ContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "terraform.tfstate")
	stateContent := `{"version": 4, "terraform_version": "1.0.0"}`
	if err := os.WriteFile(stateFile, []byte(stateContent), 0644); err != nil {
		t.Fatalf("Failed to create test state file: %v", err)
	}

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := ParseStateFile(ctx, stateFile)
	if err != context.Canceled {
		t.Errorf("ParseStateFile() with cancelled context got error = %v, want context.Canceled", err)
	}
}

func TestExtractProvider(t *testing.T) {
	tests := []struct {
		resourceType string
		want         string
	}{
		{"aws_instance", "aws"},
		{"aws_vpc", "aws"},
		{"azurerm_virtual_network", "azure"},
		{"azurerm_resource_group", "azure"},
		{"google_compute_instance", "gcp"},
		{"google_storage_bucket", "gcp"},
		{"digitalocean_droplet", "digitalocean"},
		{"digitalocean_loadbalancer", "digitalocean"},
		{"random_string", "unknown"},
		{"null_resource", "unknown"},
		{"", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.resourceType, func(t *testing.T) {
			got := extractProvider(tt.resourceType)
			if got != tt.want {
				t.Errorf("extractProvider(%s) = %s, want %s", tt.resourceType, got, tt.want)
			}
		})
	}
}

func TestParseStateFile_NonExistentFile(t *testing.T) {
	ctx := context.Background()
	_, err := ParseStateFile(ctx, "/nonexistent/path/terraform.tfstate")
	if err == nil {
		t.Error("ParseStateFile() with non-existent file should return error")
	}
}

func TestParseStateFile_ResourceIDGeneration(t *testing.T) {
	tests := []struct {
		name           string
		stateContent   string
		wantResourceID string
	}{
		{
			name: "single instance - simple ID",
			stateContent: `{
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
								"instances": [
									{
										"attributes": {
											"id": "i-12345"
										}
									}
								]
							}
						]
					}
				}
			}`,
			wantResourceID: "aws_instance.web",
		},
		{
			name: "multiple instances - indexed ID",
			stateContent: `{
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
								"instances": [
									{
										"attributes": {
											"id": "i-12345"
										}
									},
									{
										"attributes": {
											"id": "i-67890"
										}
									}
								]
							}
						]
					}
				}
			}`,
			wantResourceID: "aws_instance.web[0]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			stateFile := filepath.Join(tmpDir, "terraform.tfstate")
			if err := os.WriteFile(stateFile, []byte(tt.stateContent), 0644); err != nil {
				t.Fatalf("Failed to create test state file: %v", err)
			}

			ctx := context.Background()
			resources, err := ParseStateFile(ctx, stateFile)
			if err != nil {
				t.Fatalf("ParseStateFile() error = %v", err)
			}

			if len(resources) == 0 {
				t.Fatal("ParseStateFile() returned no resources")
			}

			if resources[0].ID != tt.wantResourceID {
				t.Errorf("Resource ID = %s, want %s", resources[0].ID, tt.wantResourceID)
			}
		})
	}
}
