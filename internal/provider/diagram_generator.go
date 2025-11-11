// Package provider implements the Terraform provider for cartography diagram generation.
// It provides both resource and data source implementations for creating infrastructure diagrams
// from Terraform state and configuration files.
package provider

import (
	"context"
	"fmt"

	"github.com/ankek/terraform-provider-cartography/internal/graph"
	"github.com/ankek/terraform-provider-cartography/internal/parser"
	"github.com/ankek/terraform-provider-cartography/internal/renderer"
	"github.com/ankek/terraform-provider-cartography/internal/validation"
)

// DiagramGenerator handles the core logic of generating diagrams.
// It is shared between the resource and data source implementations to eliminate code duplication.
// This design ensures consistency and reduces the maintenance burden by centralizing diagram generation logic.
type DiagramGenerator struct{}

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

// Generate creates a diagram from Terraform state or config files.
// This method consolidates all diagram generation logic in one place.
//
// It performs the following steps:
//  1. Validates input and output paths
//  2. Parses Terraform state or config files
//  3. Builds a resource dependency graph
//  4. Renders the diagram to the specified format
//
// Returns GenerateResult with resource count and output path, or an error if any step fails.
func (g *DiagramGenerator) Generate(ctx context.Context, cfg DiagramConfig) (*GenerateResult, error) {
	// Validate output path
	if err := validation.ValidateOutputPath(cfg.OutputPath); err != nil {
		return nil, fmt.Errorf("invalid output path: %w", err)
	}

	// Validate input paths
	if cfg.StatePath != "" {
		if err := validation.ValidateInputPath(cfg.StatePath, false); err != nil {
			return nil, fmt.Errorf("invalid state path: %w", err)
		}
	} else if cfg.ConfigPath != "" {
		if err := validation.ValidateInputPath(cfg.ConfigPath, true); err != nil {
			return nil, fmt.Errorf("invalid config path: %w", err)
		}
	}

	// Parse resources from state or config
	resources, err := g.parseResources(ctx, cfg)
	if err != nil {
		return nil, err
	}

	if len(resources) == 0 {
		return nil, fmt.Errorf("no resources found to diagram")
	}

	// Build resource dependency graph
	resourceGraph := graph.BuildGraph(ctx, resources)

	// Render diagram to file
	renderOpts := renderer.RenderOptions{
		Format:        cfg.Format,
		Direction:     cfg.Direction,
		IncludeLabels: cfg.IncludeLabels,
		Title:         cfg.Title,
		UseIcons:      cfg.UseIcons,
	}

	if err := renderer.RenderDiagram(ctx, resourceGraph, cfg.OutputPath, renderOpts); err != nil {
		return nil, fmt.Errorf("failed to render diagram: %w", err)
	}

	return &GenerateResult{
		ResourceCount: int64(len(resources)),
		OutputPath:    cfg.OutputPath,
	}, nil
}

// parseResources parses resources from either state file or config directory
func (g *DiagramGenerator) parseResources(ctx context.Context, cfg DiagramConfig) ([]parser.Resource, error) {
	// Check context before proceeding
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Determine input source
	if cfg.StatePath != "" {
		return parser.ParseStateFile(ctx, cfg.StatePath)
	}

	if cfg.ConfigPath != "" {
		return parser.ParseConfigDirectory(ctx, cfg.ConfigPath)
	}

	return nil, fmt.Errorf("either state_path or config_path must be provided")
}
