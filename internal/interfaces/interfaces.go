// Package interfaces defines interfaces for dependency injection and testing
package interfaces

import (
	"context"

	"github.com/ankek/terraform-provider-cartography/internal/graph"
	"github.com/ankek/terraform-provider-cartography/internal/parser"
	"github.com/ankek/terraform-provider-cartography/internal/renderer"
)

// Parser defines the interface for parsing Terraform configurations and state files
type Parser interface {
	// ParseStateFile parses a Terraform state file and returns resources
	ParseStateFile(ctx context.Context, path string) ([]parser.Resource, error)

	// ParseConfigDirectory parses Terraform configuration files in a directory
	ParseConfigDirectory(ctx context.Context, dirPath string) ([]parser.Resource, error)
}

// GraphBuilder defines the interface for building resource dependency graphs
type GraphBuilder interface {
	// BuildGraph creates a dependency graph from parsed resources
	BuildGraph(ctx context.Context, resources []parser.Resource) *graph.Graph
}

// DiagramRenderer defines the interface for rendering diagrams
type DiagramRenderer interface {
	// RenderDiagram generates a diagram from a graph and saves it to the output path
	RenderDiagram(ctx context.Context, g *graph.Graph, outputPath string, opts renderer.RenderOptions) error
}

// PathValidator defines the interface for validating file paths
type PathValidator interface {
	// ValidateOutputPath validates an output path for security and accessibility
	ValidateOutputPath(path string) error

	// ValidateInputPath validates an input path (state or config directory)
	ValidateInputPath(path string, mustBeDir bool) error
}

// DiagramGenerator defines the interface for generating diagrams
type DiagramGenerator interface {
	// Generate creates a diagram from Terraform state or config files
	Generate(ctx context.Context, cfg DiagramConfig) (*GenerateResult, error)
}

// DiagramConfig contains all configuration needed to generate a diagram
type DiagramConfig struct {
	StatePath     string
	ConfigPath    string
	OutputPath    string
	Format        string
	Direction     string
	IncludeLabels bool
	Title         string
	UseIcons      bool
}

// GenerateResult contains the results of diagram generation
type GenerateResult struct {
	ResourceCount int64
	OutputPath    string
}
