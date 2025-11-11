package provider

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestDiagramGenerator_Generate(t *testing.T) {
	// Create temporary directory for test outputs
	tmpDir := t.TempDir()

	// Create a test state file
	stateFile := filepath.Join(tmpDir, "terraform.tfstate")
	stateContent := `{
		"version": 4,
		"terraform_version": "1.0.0",
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
						}
					}
				]
			}
		]
	}`
	if err := os.WriteFile(stateFile, []byte(stateContent), 0644); err != nil {
		t.Fatalf("Failed to create test state file: %v", err)
	}

	generator := &DiagramGenerator{}
	ctx := context.Background()

	tests := []struct {
		name    string
		config  DiagramConfig
		wantErr bool
	}{
		{
			name: "valid state file",
			config: DiagramConfig{
				StatePath:     stateFile,
				OutputPath:    filepath.Join(tmpDir, "diagram.svg"),
				Format:        "svg",
				Direction:     "TB",
				IncludeLabels: true,
				UseIcons:      false,
			},
			wantErr: false,
		},
		{
			name: "missing input",
			config: DiagramConfig{
				OutputPath:    filepath.Join(tmpDir, "diagram.svg"),
				Format:        "svg",
				Direction:     "TB",
				IncludeLabels: true,
			},
			wantErr: true,
		},
		{
			name: "invalid output path",
			config: DiagramConfig{
				StatePath:  stateFile,
				OutputPath: "/nonexistent/directory/diagram.svg",
				Format:     "svg",
			},
			wantErr: true,
		},
		{
			name: "non-existent state file",
			config: DiagramConfig{
				StatePath:  "/nonexistent/state.tfstate",
				OutputPath: filepath.Join(tmpDir, "diagram.svg"),
				Format:     "svg",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := generator.Generate(ctx, tt.config)

			if (err != nil) != tt.wantErr {
				t.Errorf("Generate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if result == nil {
					t.Error("Generate() returned nil result for successful generation")
					return
				}

				if result.ResourceCount <= 0 {
					t.Errorf("Generate() ResourceCount = %d, want > 0", result.ResourceCount)
				}

				if result.OutputPath != tt.config.OutputPath {
					t.Errorf("Generate() OutputPath = %v, want %v", result.OutputPath, tt.config.OutputPath)
				}

				// Verify output file was created
				if _, err := os.Stat(result.OutputPath); os.IsNotExist(err) {
					t.Errorf("Generate() did not create output file at %s", result.OutputPath)
				}
			}
		})
	}
}

func TestDiagramGenerator_Generate_ContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test state file
	stateFile := filepath.Join(tmpDir, "terraform.tfstate")
	stateContent := `{
		"version": 4,
		"terraform_version": "1.0.0",
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
						}
					}
				]
			}
		]
	}`
	if err := os.WriteFile(stateFile, []byte(stateContent), 0644); err != nil {
		t.Fatalf("Failed to create test state file: %v", err)
	}

	generator := &DiagramGenerator{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	config := DiagramConfig{
		StatePath:  stateFile,
		OutputPath: filepath.Join(tmpDir, "diagram.svg"),
		Format:     "svg",
		Direction:  "TB",
	}

	_, err := generator.Generate(ctx, config)

	// Should get context canceled error
	if err == nil {
		t.Error("Generate() should fail when context is cancelled")
	}
}

func TestParseResources(t *testing.T) {
	tmpDir := t.TempDir()
	generator := &DiagramGenerator{}
	ctx := context.Background()

	// Create test state file with actual resources
	stateFile := filepath.Join(tmpDir, "terraform.tfstate")
	stateContent := `{
		"version": 4,
		"terraform_version": "1.0.0",
		"resources": [
			{
				"mode": "managed",
				"type": "aws_instance",
				"name": "web",
				"provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
				"instances": [
					{
						"attributes": {
							"id": "i-test",
							"instance_type": "t2.micro"
						}
					}
				]
			}
		]
	}`
	if err := os.WriteFile(stateFile, []byte(stateContent), 0644); err != nil {
		t.Fatalf("Failed to create test state file: %v", err)
	}

	// Create test config directory
	configDir := filepath.Join(tmpDir, "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}

	// Create a simple .tf file
	tfFile := filepath.Join(configDir, "main.tf")
	tfContent := `
resource "aws_instance" "web" {
  ami           = "ami-12345"
  instance_type = "t2.micro"
}
`
	if err := os.WriteFile(tfFile, []byte(tfContent), 0644); err != nil {
		t.Fatalf("Failed to create .tf file: %v", err)
	}

	tests := []struct {
		name    string
		config  DiagramConfig
		wantErr bool
	}{
		{
			name: "parse state file",
			config: DiagramConfig{
				StatePath: stateFile,
			},
			wantErr: false,
		},
		{
			name: "parse config directory",
			config: DiagramConfig{
				ConfigPath: configDir,
			},
			wantErr: false,
		},
		{
			name:    "no input",
			config:  DiagramConfig{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := generator.parseResources(ctx, tt.config)

			if (err != nil) != tt.wantErr {
				t.Errorf("parseResources() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestDiagramConfig_Validation(t *testing.T) {
	tmpDir := t.TempDir()
	generator := &DiagramGenerator{}
	ctx := context.Background()

	// Create valid state file
	stateFile := filepath.Join(tmpDir, "terraform.tfstate")
	stateContent := `{
		"version": 4,
		"terraform_version": "1.0.0",
		"resources": [
			{
				"mode": "managed",
				"type": "aws_instance",
				"name": "web",
				"provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
				"instances": [{"attributes": {"id": "i-12345"}}]
			}
		]
	}`
	if err := os.WriteFile(stateFile, []byte(stateContent), 0644); err != nil {
		t.Fatalf("Failed to create test state file: %v", err)
	}

	tests := []struct {
		name    string
		config  DiagramConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid SVG format",
			config: DiagramConfig{
				StatePath:  stateFile,
				OutputPath: filepath.Join(tmpDir, "test.svg"),
				Format:     "svg",
				Direction:  "TB",
			},
			wantErr: false,
		},
		{
			name: "all directions",
			config: DiagramConfig{
				StatePath:  stateFile,
				OutputPath: filepath.Join(tmpDir, "test.svg"),
				Format:     "svg",
				Direction:  "BT",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := generator.Generate(ctx, tt.config)

			if (err != nil) != tt.wantErr {
				t.Errorf("Generate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
