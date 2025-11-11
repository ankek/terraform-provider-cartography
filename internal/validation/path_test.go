package validation

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateOutputPath(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	tests := []struct {
		name    string
		path    string
		wantErr bool
		setup   func() string
	}{
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
		},
		{
			name:    "valid path in temp dir",
			wantErr: false,
			setup: func() string {
				return filepath.Join(tmpDir, "test.svg")
			},
		},
		{
			name:    "path traversal attempt with ..",
			path:    tmpDir + "/../../../etc/passwd",
			wantErr: true,
		},
		{
			name:    "path in non-existent directory",
			path:    "/nonexistent/directory/file.svg",
			wantErr: true,
		},
		{
			name:    "valid nested path",
			wantErr: false,
			setup: func() string {
				nested := filepath.Join(tmpDir, "nested", "dir")
				os.MkdirAll(nested, 0755)
				return filepath.Join(nested, "diagram.svg")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.path
			if tt.setup != nil {
				path = tt.setup()
			}

			err := ValidateOutputPath(path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateOutputPath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateInputPath(t *testing.T) {
	// Create temporary test structure
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.tfstate")
	testDir := filepath.Join(tmpDir, "config")

	// Create test file
	if err := os.WriteFile(testFile, []byte("{}"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create test directory
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	tests := []struct {
		name      string
		path      string
		mustBeDir bool
		wantErr   bool
	}{
		{
			name:      "empty path",
			path:      "",
			mustBeDir: false,
			wantErr:   true,
		},
		{
			name:      "valid file when file expected",
			path:      testFile,
			mustBeDir: false,
			wantErr:   false,
		},
		{
			name:      "valid directory when directory expected",
			path:      testDir,
			mustBeDir: true,
			wantErr:   false,
		},
		{
			name:      "file when directory expected",
			path:      testFile,
			mustBeDir: true,
			wantErr:   true,
		},
		{
			name:      "directory when file expected",
			path:      testDir,
			mustBeDir: false,
			wantErr:   true,
		},
		{
			name:      "non-existent path",
			path:      "/nonexistent/path",
			mustBeDir: false,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateInputPath(tt.path, tt.mustBeDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateInputPath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateOutputPath_Permissions(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Skipping permission test when running as root")
	}

	// Skip on Windows - permissions work differently
	if os.PathSeparator == '\\' {
		t.Skip("Skipping permission test on Windows")
	}

	// Create a read-only directory
	tmpDir := t.TempDir()
	readOnlyDir := filepath.Join(tmpDir, "readonly")
	if err := os.MkdirAll(readOnlyDir, 0555); err != nil {
		t.Fatalf("Failed to create read-only directory: %v", err)
	}
	defer os.Chmod(readOnlyDir, 0755) // Restore permissions for cleanup

	// Try to validate a path in the read-only directory
	testPath := filepath.Join(readOnlyDir, "test.svg")
	err := ValidateOutputPath(testPath)

	if err == nil {
		t.Error("ValidateOutputPath() should fail for read-only directory")
	}
}
