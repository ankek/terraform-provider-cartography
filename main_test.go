package main

import (
	"testing"
)

func TestVersion(t *testing.T) {
	// Test that version variable exists and has a default value
	if version == "" {
		t.Error("version should not be empty")
	}

	// Default version should be "dev"
	if version != "dev" {
		t.Logf("version = %s (expected 'dev' but may be set by build)", version)
	}
}

func TestMainPackage(t *testing.T) {
	// This test verifies that the main package compiles correctly
	// and that all imports are valid
	t.Log("Main package compiles successfully")
}
