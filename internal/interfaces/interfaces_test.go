package interfaces

import (
	"testing"
)

func TestDiagramConfigStruct(t *testing.T) {
	// Test that DiagramConfig can be created and has all fields
	cfg := DiagramConfig{
		StatePath:     "/path/to/state.tfstate",
		ConfigPath:    "/path/to/config",
		OutputPath:    "/path/to/output.svg",
		Format:        "svg",
		Direction:     "TB",
		IncludeLabels: true,
		Title:         "Test Diagram",
		UseIcons:      true,
	}

	if cfg.StatePath != "/path/to/state.tfstate" {
		t.Errorf("Expected StatePath '/path/to/state.tfstate', got '%s'", cfg.StatePath)
	}
	if cfg.Format != "svg" {
		t.Errorf("Expected Format 'svg', got '%s'", cfg.Format)
	}
	if !cfg.IncludeLabels {
		t.Error("Expected IncludeLabels to be true")
	}
	if !cfg.UseIcons {
		t.Error("Expected UseIcons to be true")
	}
}

func TestGenerateResultStruct(t *testing.T) {
	// Test that GenerateResult can be created and has all fields
	result := GenerateResult{
		ResourceCount: 42,
		OutputPath:    "/path/to/output.svg",
	}

	if result.ResourceCount != 42 {
		t.Errorf("Expected ResourceCount 42, got %d", result.ResourceCount)
	}
	if result.OutputPath != "/path/to/output.svg" {
		t.Errorf("Expected OutputPath '/path/to/output.svg', got '%s'", result.OutputPath)
	}
}

func TestInterfacesAreDefined(t *testing.T) {
	// This test verifies that all interfaces are properly defined
	// by checking that we can reference them without compile errors

	// Parser interface
	var _ Parser

	// GraphBuilder interface
	var _ GraphBuilder

	// DiagramRenderer interface
	var _ DiagramRenderer

	// PathValidator interface
	var _ PathValidator

	// DiagramGenerator interface
	var _ DiagramGenerator

	t.Log("All interfaces are properly defined")
}
