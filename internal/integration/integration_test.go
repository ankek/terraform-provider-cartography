package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ankek/terraform-provider-cartography/internal/graph"
	"github.com/ankek/terraform-provider-cartography/internal/parser"
	"github.com/ankek/terraform-provider-cartography/internal/provider"
	"github.com/ankek/terraform-provider-cartography/internal/renderer"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestFullPipeline tests the complete workflow from state file to diagram
func TestFullPipeline(t *testing.T) {
	tests := []struct {
		name          string
		stateContent  string
		wantNodes     int
		wantEdges     int
		outputFormat  string
		includeLabels bool
		useIcons      bool
	}{
		{
			name: "AWS infrastructure",
			stateContent: `{
				"version": 4,
				"terraform_version": "1.0.0",
				"values": {
					"root_module": {
						"resources": [
							{
								"mode": "managed",
								"type": "aws_vpc",
								"name": "main",
								"provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
								"instances": [{
									"attributes": {
										"id": "vpc-12345",
										"cidr_block": "10.0.0.0/16"
									}
								}]
							},
							{
								"mode": "managed",
								"type": "aws_subnet",
								"name": "public",
								"provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
								"instances": [{
									"attributes": {
										"id": "subnet-12345",
										"vpc_id": "vpc-12345"
									},
									"dependencies": ["aws_vpc.main"]
								}]
							},
							{
								"mode": "managed",
								"type": "aws_instance",
								"name": "web",
								"provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
								"instances": [{
									"attributes": {
										"id": "i-12345",
										"instance_type": "t2.micro",
										"subnet_id": "subnet-12345"
									},
									"dependencies": ["aws_subnet.public"]
								}]
							}
						]
					}
				}
			}`,
			wantNodes:     3,
			wantEdges:     2,
			outputFormat:  "svg",
			includeLabels: true,
			useIcons:      false,
		},
		{
			name: "Azure infrastructure",
			stateContent: `{
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
								"instances": [{
									"attributes": {
										"id": "/subscriptions/xxx/resourceGroups/test-rg",
										"name": "test-rg",
										"location": "eastus"
									}
								}]
							},
							{
								"mode": "managed",
								"type": "azurerm_virtual_network",
								"name": "vnet",
								"provider": "provider[\"registry.terraform.io/hashicorp/azurerm\"]",
								"instances": [{
									"attributes": {
										"id": "/subscriptions/xxx/resourceGroups/test-rg/providers/Microsoft.Network/virtualNetworks/test-vnet",
										"resource_group_name": "test-rg"
									},
									"dependencies": ["azurerm_resource_group.rg"]
								}]
							}
						]
					}
				}
			}`,
			wantNodes:     2,
			wantEdges:     1,
			outputFormat:  "svg",
			includeLabels: true,
			useIcons:      false,
		},
		{
			name: "multi-cloud infrastructure",
			stateContent: `{
				"version": 4,
				"terraform_version": "1.0.0",
				"values": {
					"root_module": {
						"resources": [
							{
								"mode": "managed",
								"type": "aws_s3_bucket",
								"name": "storage",
								"provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
								"instances": [{"attributes": {"id": "my-bucket"}}]
							},
							{
								"mode": "managed",
								"type": "azurerm_storage_account",
								"name": "storage",
								"provider": "provider[\"registry.terraform.io/hashicorp/azurerm\"]",
								"instances": [{"attributes": {"id": "/subscriptions/xxx/resourceGroups/rg/providers/Microsoft.Storage/storageAccounts/mystorageaccount"}}]
							},
							{
								"mode": "managed",
								"type": "google_storage_bucket",
								"name": "storage",
								"provider": "provider[\"registry.terraform.io/hashicorp/google\"]",
								"instances": [{"attributes": {"id": "my-gcs-bucket"}}]
							}
						]
					}
				}
			}`,
			wantNodes:     3,
			wantEdges:     0,
			outputFormat:  "svg",
			includeLabels: true,
			useIcons:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			// Step 1: Create state file
			stateFile := filepath.Join(tmpDir, "terraform.tfstate")
			if err := os.WriteFile(stateFile, []byte(tt.stateContent), 0644); err != nil {
				t.Fatalf("Failed to create state file: %v", err)
			}

			ctx := context.Background()

			// Step 2: Parse state file
			resources, err := parser.ParseStateFile(ctx, stateFile)
			if err != nil {
				t.Fatalf("ParseStateFile() error = %v", err)
			}

			if len(resources) != tt.wantNodes {
				t.Errorf("ParseStateFile() got %d resources, want %d", len(resources), tt.wantNodes)
			}

			// Step 3: Build graph
			g := graph.BuildGraph(ctx, resources)

			if len(g.Nodes) != tt.wantNodes {
				t.Errorf("BuildGraph() got %d nodes, want %d", len(g.Nodes), tt.wantNodes)
			}

			if len(g.Edges) < tt.wantEdges {
				t.Errorf("BuildGraph() got %d edges, want at least %d", len(g.Edges), tt.wantEdges)
			}

			// Step 4: Render diagram
			outputPath := filepath.Join(tmpDir, "diagram."+tt.outputFormat)
			opts := renderer.RenderOptions{
				Format:        tt.outputFormat,
				Direction:     "TB",
				IncludeLabels: tt.includeLabels,
				Title:         "Test Infrastructure",
				UseIcons:      tt.useIcons,
			}

			if err := renderer.RenderDiagram(ctx, g, outputPath, opts); err != nil {
				t.Fatalf("RenderDiagram() error = %v", err)
			}

			// Step 5: Verify output file
			if _, err := os.Stat(outputPath); os.IsNotExist(err) {
				t.Error("RenderDiagram() did not create output file")
			}

			// Verify file has content
			content, err := os.ReadFile(outputPath)
			if err != nil {
				t.Fatalf("Failed to read output file: %v", err)
			}
			if len(content) == 0 {
				t.Error("Output file is empty")
			}

			// For SVG, verify basic structure
			if tt.outputFormat == "svg" {
				contentStr := string(content)
				if len(contentStr) < 100 {
					t.Error("SVG content is too short")
				}
			}
		})
	}
}

// TestDiagramGeneratorEndToEnd tests the DiagramGenerator with real state files
func TestDiagramGeneratorEndToEnd(t *testing.T) {
	tmpDir := t.TempDir()

	stateContent := `{
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
						"instances": [{
							"attributes": {
								"id": "123456",
								"name": "web-server",
								"size": "s-1vcpu-1gb"
							}
						}]
					},
					{
						"mode": "managed",
						"type": "digitalocean_loadbalancer",
						"name": "lb",
						"provider": "provider[\"registry.terraform.io/digitalocean/digitalocean\"]",
						"instances": [{
							"attributes": {
								"id": "lb-123",
								"name": "web-lb"
							},
							"dependencies": ["digitalocean_droplet.web"]
						}]
					}
				]
			}
		}
	}`

	stateFile := filepath.Join(tmpDir, "terraform.tfstate")
	if err := os.WriteFile(stateFile, []byte(stateContent), 0644); err != nil {
		t.Fatalf("Failed to create state file: %v", err)
	}

	outputPath := filepath.Join(tmpDir, "infrastructure.svg")

	gen := &provider.DiagramGenerator{}
	cfg := provider.DiagramConfig{
		StatePath:     stateFile,
		OutputPath:    outputPath,
		Format:        "svg",
		Direction:     "TB",
		IncludeLabels: true,
		Title:         "DigitalOcean Infrastructure",
		UseIcons:      false,
	}

	ctx := context.Background()
	result, err := gen.Generate(ctx, cfg)

	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if result.ResourceCount != 2 {
		t.Errorf("Generate() resource count = %d, want 2", result.ResourceCount)
	}

	if result.OutputPath != outputPath {
		t.Errorf("Generate() output path = %s, want %s", result.OutputPath, outputPath)
	}

	// Verify file exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("Generate() did not create output file")
	}
}

// TestLoadResourcesWithBackend tests state loading with backend configuration
func TestLoadResourcesWithBackend(t *testing.T) {
	tmpDir := t.TempDir()

	// Create backend configuration
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

	// Create state file
	stateFile := filepath.Join(tmpDir, "custom.tfstate")
	stateContent := `{
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
						"instances": [{
							"attributes": {
								"id": "projects/my-project/zones/us-central1-a/instances/my-vm",
								"name": "my-vm"
							}
						}]
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
	resources, err := provider.LoadResources(ctx, nil, types.StringNull(), configPath)

	if err != nil {
		t.Fatalf("LoadResources() error = %v", err)
	}

	if len(resources) != 1 {
		t.Errorf("LoadResources() got %d resources, want 1", len(resources))
	}

	if resources[0].Provider != "gcp" {
		t.Errorf("LoadResources() provider = %s, want gcp", resources[0].Provider)
	}
}

// TestConfigParsingEndToEnd tests parsing Terraform configuration files
func TestConfigParsingEndToEnd(t *testing.T) {
	tmpDir := t.TempDir()

	// Create multiple .tf files
	files := map[string]string{
		"main.tf": `
resource "aws_vpc" "main" {
  cidr_block = "10.0.0.0/16"
  tags = {
    Name = "main-vpc"
  }
}
`,
		"compute.tf": `
resource "aws_instance" "web" {
  ami           = "ami-12345"
  instance_type = "t2.micro"
  vpc_id        = aws_vpc.main.id
}

resource "aws_instance" "db" {
  ami           = "ami-67890"
  instance_type = "t2.small"
  vpc_id        = aws_vpc.main.id
}
`,
	}

	for filename, content := range files {
		filePath := filepath.Join(tmpDir, filename)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", filename, err)
		}
	}

	ctx := context.Background()

	// Parse config directory
	resources, err := parser.ParseConfigDirectory(ctx, tmpDir)
	if err != nil {
		t.Fatalf("ParseConfigDirectory() error = %v", err)
	}

	if len(resources) != 3 {
		t.Errorf("ParseConfigDirectory() got %d resources, want 3", len(resources))
	}

	// Build and render diagram
	g := graph.BuildGraph(ctx, resources)

	outputPath := filepath.Join(tmpDir, "config-diagram.svg")
	opts := renderer.RenderOptions{
		Format:        "svg",
		Direction:     "TB",
		IncludeLabels: true,
		Title:         "Configuration Diagram",
		UseIcons:      false,
	}

	if err := renderer.RenderDiagram(ctx, g, outputPath, opts); err != nil {
		t.Fatalf("RenderDiagram() error = %v", err)
	}

	// Verify output
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("Diagram file was not created")
	}
}
