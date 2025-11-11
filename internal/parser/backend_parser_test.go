package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseBackendConfig(t *testing.T) {
	tests := []struct {
		name            string
		files           map[string]string
		wantBackendType string
		wantConfig      map[string]interface{}
		wantErr         bool
	}{
		{
			name: "local backend",
			files: map[string]string{
				"backend.tf": `
terraform {
  backend "local" {
    path = "terraform.tfstate"
  }
}
`,
			},
			wantBackendType: "local",
			wantConfig: map[string]interface{}{
				"path": "terraform.tfstate",
			},
			wantErr: false,
		},
		{
			name: "s3 backend",
			files: map[string]string{
				"backend.tf": `
terraform {
  backend "s3" {
    bucket = "my-terraform-state"
    key    = "prod/terraform.tfstate"
    region = "us-east-1"
  }
}
`,
			},
			wantBackendType: "s3",
			wantConfig: map[string]interface{}{
				"bucket": "my-terraform-state",
				"key":    "prod/terraform.tfstate",
				"region": "us-east-1",
			},
			wantErr: false,
		},
		{
			name: "azurerm backend",
			files: map[string]string{
				"backend.tf": `
terraform {
  backend "azurerm" {
    storage_account_name = "mystorageaccount"
    container_name       = "tfstate"
    key                  = "prod.terraform.tfstate"
  }
}
`,
			},
			wantBackendType: "azurerm",
			wantConfig: map[string]interface{}{
				"storage_account_name": "mystorageaccount",
				"container_name":       "tfstate",
				"key":                  "prod.terraform.tfstate",
			},
			wantErr: false,
		},
		{
			name: "remote backend (terraform cloud)",
			files: map[string]string{
				"backend.tf": `
terraform {
  backend "remote" {
    hostname     = "app.terraform.io"
    organization = "my-org"
    
    workspaces {
      name = "my-workspace"
    }
  }
}
`,
			},
			wantBackendType: "remote",
			wantErr:         false,
		},
		{
			name: "gcs backend",
			files: map[string]string{
				"backend.tf": `
terraform {
  backend "gcs" {
    bucket = "my-terraform-state"
    prefix = "prod"
  }
}
`,
			},
			wantBackendType: "gcs",
			wantConfig: map[string]interface{}{
				"bucket": "my-terraform-state",
				"prefix": "prod",
			},
			wantErr: false,
		},
		{
			name: "no backend - defaults to local",
			files: map[string]string{
				"main.tf": `
resource "aws_instance" "web" {
  ami = "ami-12345"
}
`,
			},
			wantBackendType: "local",
			wantErr:         false,
		},
		{
			name: "multiple terraform blocks - use first backend",
			files: map[string]string{
				"backend.tf": `
terraform {
  backend "s3" {
    bucket = "my-state"
    key    = "terraform.tfstate"
  }
}
`,
				"other.tf": `
terraform {
  required_version = ">= 1.0"
}
`,
			},
			wantBackendType: "s3",
			wantErr:         false,
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

			backend, err := ParseBackendConfig(tmpDir)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseBackendConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				if backend.Type != tt.wantBackendType {
					t.Errorf("ParseBackendConfig() backend type = %s, want %s", backend.Type, tt.wantBackendType)
				}

				if backend.WorkingDir != tmpDir {
					t.Errorf("ParseBackendConfig() working dir = %s, want %s", backend.WorkingDir, tmpDir)
				}

				// Check specific config values if provided
				for key, expectedValue := range tt.wantConfig {
					if actualValue, ok := backend.Config[key]; ok {
						if actualValue != expectedValue {
							t.Errorf("Backend config[%s] = %v, want %v", key, actualValue, expectedValue)
						}
					} else {
						t.Errorf("Backend config missing key: %s", key)
					}
				}
			}
		})
	}
}

func TestParseBackendConfig_InvalidDirectory(t *testing.T) {
	_, err := ParseBackendConfig("/nonexistent/directory")
	if err == nil {
		t.Error("ParseBackendConfig() with non-existent directory should return error")
	}
}

func TestGetStatePath(t *testing.T) {
	tests := []struct {
		name       string
		setupFiles map[string]string
		backend    *BackendConfig
		wantErr    bool
	}{
		{
			name: "local backend with path",
			setupFiles: map[string]string{
				"terraform.tfstate": `{"version": 4}`,
			},
			backend: &BackendConfig{
				Type: "local",
				Config: map[string]interface{}{
					"path": "terraform.tfstate",
				},
				WorkingDir: "",
			},
			wantErr: false,
		},
		{
			name: "local backend without path - default",
			setupFiles: map[string]string{
				"terraform.tfstate": `{"version": 4}`,
			},
			backend: &BackendConfig{
				Type:       "local",
				Config:     map[string]interface{}{},
				WorkingDir: "",
			},
			wantErr: false,
		},
		{
			name:       "remote backend - should error",
			setupFiles: map[string]string{},
			backend: &BackendConfig{
				Type: "s3",
				Config: map[string]interface{}{
					"bucket": "my-bucket",
				},
				WorkingDir: "",
			},
			wantErr: true,
		},
		{
			name:       "local backend - file not found",
			setupFiles: map[string]string{},
			backend: &BackendConfig{
				Type:       "local",
				Config:     map[string]interface{}{},
				WorkingDir: "",
			},
			wantErr: true,
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

			got, err := GetStatePath(tt.backend)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetStatePath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify returned path exists
				if _, err := os.Stat(got); os.IsNotExist(err) {
					t.Errorf("GetStatePath() returned non-existent path: %s", got)
				}
			}
		})
	}
}

func TestAutoDetectStatePath(t *testing.T) {
	tests := []struct {
		name      string
		files     []string
		wantFound bool
	}{
		{
			name:      "terraform.tfstate exists",
			files:     []string{"terraform.tfstate"},
			wantFound: true,
		},
		{
			name:      ".terraform/terraform.tfstate exists",
			files:     []string{".terraform/terraform.tfstate"},
			wantFound: true,
		},
		{
			name:      "no state files",
			files:     []string{"main.tf", "README.md"},
			wantFound: false,
		},
		{
			name:      "prefer terraform.tfstate over .terraform location",
			files:     []string{"terraform.tfstate", ".terraform/terraform.tfstate"},
			wantFound: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			// Create test files
			for _, filename := range tt.files {
				filePath := filepath.Join(tmpDir, filename)
				// Create directory if needed
				dir := filepath.Dir(filePath)
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatalf("Failed to create directory %s: %v", dir, err)
				}
				if err := os.WriteFile(filePath, []byte("{}"), 0644); err != nil {
					t.Fatalf("Failed to create test file %s: %v", filename, err)
				}
			}

			got, err := AutoDetectStatePath(tmpDir)

			if tt.wantFound && err != nil {
				t.Errorf("AutoDetectStatePath() unexpected error: %v", err)
			}

			if !tt.wantFound && err == nil {
				t.Error("AutoDetectStatePath() should return error when no state file found")
			}

			if tt.wantFound && err == nil {
				if !filepath.IsAbs(got) {
					t.Errorf("AutoDetectStatePath() returned relative path: %s", got)
				}
				// Verify the file exists
				if _, err := os.Stat(got); os.IsNotExist(err) {
					t.Errorf("AutoDetectStatePath() returned non-existent path: %s", got)
				}
			}
		})
	}
}

func TestBackendType_Constants(t *testing.T) {
	// Verify backend type constants are defined correctly
	backends := []BackendType{
		BackendTypeLocal,
		BackendTypeRemote,
		BackendTypeS3,
		BackendTypeAzureRM,
		BackendTypeGCS,
		BackendTypeHTTP,
		BackendTypeConsul,
		BackendTypeEtcdV3,
		BackendTypePg,
	}

	for _, backend := range backends {
		if string(backend) == "" {
			t.Errorf("Backend type should not be empty: %v", backend)
		}
	}
}
