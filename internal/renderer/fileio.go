package renderer

import (
	"fmt"
	"os"
)

// writeFile writes data to a file
func writeFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0644)
}

// readFile reads data from a file
func readFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// createFile creates a new file for writing
func createFile(path string) (*os.File, error) {
	file, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("failed to create file %s: %w", path, err)
	}
	return file, nil
}
