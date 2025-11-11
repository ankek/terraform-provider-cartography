package parser

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestParseConfigDirectory(t *testing.T) {
	tests := []struct {
		name          string
		files         map[string]string
		wantResources int
		wantErr       bool
	}{
		{
			name: "single file with resources",
			files: map[string]string{
				"main.tf": `
resource "aws_instance" "web" {
  ami           = "ami-12345"
  instance_type = "t2.micro"
}

resource "aws_vpc" "main" {
  cidr_block = "10.0.0.0/16"
}
`,
			},
			wantResources: 2,
			wantErr:       false,
		},
		{
			name: "multiple files",
			files: map[string]string{
				"main.tf": `
resource "aws_instance" "web" {
  ami           = "ami-12345"
  instance_type = "t2.micro"
}
`,
				"network.tf": `
resource "aws_vpc" "main" {
  cidr_block = "10.0.0.0/16"
}

resource "aws_subnet" "public" {
  vpc_id     = aws_vpc.main.id
  cidr_block = "10.0.1.0/24"
}
`,
			},
			wantResources: 3,
			wantErr:       false,
		},
		{
			name: "empty directory",
			files: map[string]string{
				"README.md": "# Test",
			},
			wantResources: 0,
			wantErr:       false,
		},
		{
			name: "invalid HCL",
			files: map[string]string{
				"main.tf": `
resource "aws_instance" "web" {
  # Invalid - missing closing brace
`,
			},
			wantResources: 0,
			wantErr:       true,
		},
		{
			name: "mixed valid and non-tf files",
			files: map[string]string{
				"main.tf": `
resource "azurerm_resource_group" "rg" {
  name     = "example-rg"
  location = "eastus"
}
`,
				"variables.tf": `
variable "location" {
  default = "eastus"
}
`,
				"outputs.tf": `
output "rg_name" {
  value = azurerm_resource_group.rg.name
}
`,
				"README.md": "Documentation",
			},
			wantResources: 1,
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			// Create test files
			for filename, content := range tt.files {
				filePath := filepath.Join(tmpDir, filename)
				if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
					t.Fatalf("Failed to create test file %s: %v", filename, err)
				}
			}

			ctx := context.Background()
			resources, err := ParseConfigDirectory(ctx, tmpDir)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseConfigDirectory() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil && len(resources) != tt.wantResources {
				t.Errorf("ParseConfigDirectory() got %d resources, want %d", len(resources), tt.wantResources)
			}
		})
	}
}

func TestParseConfigDirectory_ContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()
	tfFile := filepath.Join(tmpDir, "main.tf")
	content := `resource "aws_instance" "web" { ami = "ami-12345" }`
	if err := os.WriteFile(tfFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := ParseConfigDirectory(ctx, tmpDir)
	if err != context.Canceled {
		t.Errorf("ParseConfigDirectory() with cancelled context got error = %v, want context.Canceled", err)
	}
}

func TestParseConfigDirectory_NonExistentDirectory(t *testing.T) {
	ctx := context.Background()
	_, err := ParseConfigDirectory(ctx, "/nonexistent/directory")
	if err == nil {
		t.Error("ParseConfigDirectory() with non-existent directory should return error")
	}
}

func TestParseConfigDirectory_WithDependencies(t *testing.T) {
	tmpDir := t.TempDir()
	tfFile := filepath.Join(tmpDir, "main.tf")
	content := `
resource "aws_vpc" "main" {
  cidr_block = "10.0.0.0/16"
}

resource "aws_subnet" "public" {
  vpc_id     = aws_vpc.main.id
  cidr_block = "10.0.1.0/24"
}

resource "aws_instance" "web" {
  ami           = "ami-12345"
  instance_type = "t2.micro"
  subnet_id     = aws_subnet.public.id
  
  depends_on = [aws_vpc.main]
}
`
	if err := os.WriteFile(tfFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	ctx := context.Background()
	resources, err := ParseConfigDirectory(ctx, tmpDir)
	if err != nil {
		t.Fatalf("ParseConfigDirectory() error = %v", err)
	}

	if len(resources) != 3 {
		t.Errorf("ParseConfigDirectory() got %d resources, want 3", len(resources))
	}

	// Check that dependencies were extracted
	hasInstanceResource := false
	for _, res := range resources {
		if res.Type == "aws_instance" && res.Name == "web" {
			hasInstanceResource = true
			if len(res.Dependencies) == 0 {
				t.Error("aws_instance.web should have dependencies")
			}
		}
	}

	if !hasInstanceResource {
		t.Error("aws_instance.web not found in parsed resources")
	}
}

func TestParseConfigDirectory_MultiCloudProviders(t *testing.T) {
	tmpDir := t.TempDir()

	files := map[string]string{
		"aws.tf": `
resource "aws_instance" "web" {
  ami = "ami-12345"
}
`,
		"azure.tf": `
resource "azurerm_virtual_network" "vnet" {
  name = "example-vnet"
}
`,
		"gcp.tf": `
resource "google_compute_instance" "vm" {
  name = "example-vm"
}
`,
		"digitalocean.tf": `
resource "digitalocean_droplet" "web" {
  name = "example-droplet"
}
`,
	}

	for filename, content := range files {
		filePath := filepath.Join(tmpDir, filename)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}

	ctx := context.Background()
	resources, err := ParseConfigDirectory(ctx, tmpDir)
	if err != nil {
		t.Fatalf("ParseConfigDirectory() error = %v", err)
	}

	if len(resources) != 4 {
		t.Errorf("ParseConfigDirectory() got %d resources, want 4", len(resources))
	}

	// Verify providers were extracted correctly
	providerCounts := make(map[string]int)
	for _, res := range resources {
		providerCounts[res.Provider]++
	}

	expectedProviders := map[string]int{
		"aws":          1,
		"azure":        1,
		"gcp":          1,
		"digitalocean": 1,
	}

	for provider, expectedCount := range expectedProviders {
		if providerCounts[provider] != expectedCount {
			t.Errorf("Expected %d resources for %s provider, got %d", expectedCount, provider, providerCounts[provider])
		}
	}
}
