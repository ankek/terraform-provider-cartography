// Package validation provides security and safety validation utilities for file paths and user input.
// It includes protection against path traversal attacks and validation of file system permissions.
package validation

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ValidateOutputPath validates an output path for security and accessibility
// Returns error if path is invalid, contains path traversal attempts, or is not writable
func ValidateOutputPath(outputPath string) error {
	if outputPath == "" {
		return fmt.Errorf("output path cannot be empty")
	}

	// Clean the path to resolve any . or .. components
	cleanPath := filepath.Clean(outputPath)

	// Check for path traversal attempts
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("path traversal detected in output path: %s", outputPath)
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	// Get the directory
	dir := filepath.Dir(absPath)

	// Check if directory exists
	dirInfo, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("output directory does not exist: %s", dir)
		}
		return fmt.Errorf("failed to access output directory: %w", err)
	}

	// Ensure it's a directory
	if !dirInfo.IsDir() {
		return fmt.Errorf("output path parent is not a directory: %s", dir)
	}

	// Check if directory is writable by attempting to create a temp file
	testFile := filepath.Join(dir, ".cartography_write_test")
	f, err := os.OpenFile(testFile, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0600)
	if err != nil {
		return fmt.Errorf("output directory is not writable: %s: %w", dir, err)
	}
	f.Close()
	os.Remove(testFile) // Clean up test file

	return nil
}

// ValidateInputPath validates an input path (state or config directory)
// Returns error if path doesn't exist or is not accessible
func ValidateInputPath(inputPath string, mustBeDir bool) error {
	if inputPath == "" {
		return fmt.Errorf("input path cannot be empty")
	}

	// Clean the path
	cleanPath := filepath.Clean(inputPath)

	// Check for path traversal in relative context
	if strings.Contains(cleanPath, "..") && !filepath.IsAbs(inputPath) {
		// Allow absolute paths with .. after cleaning, but be careful with relative paths
		return fmt.Errorf("potentially unsafe path detected: %s", inputPath)
	}

	// Check if path exists
	info, err := os.Stat(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("input path does not exist: %s", cleanPath)
		}
		return fmt.Errorf("failed to access input path: %w", err)
	}

	// If must be directory, verify
	if mustBeDir && !info.IsDir() {
		return fmt.Errorf("input path must be a directory: %s", cleanPath)
	}

	// If must be file, verify
	if !mustBeDir && info.IsDir() {
		return fmt.Errorf("input path must be a file: %s", cleanPath)
	}

	return nil
}
