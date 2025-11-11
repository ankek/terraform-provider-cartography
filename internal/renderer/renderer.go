// Package renderer provides functionality for rendering infrastructure diagrams
// from Terraform resource graphs. It supports multiple output formats (SVG, PNG, JPEG)
// and includes professional styling, icon support, and layout algorithms.
package renderer

import (
	"context"

	"github.com/ankek/terraform-provider-cartography/internal/graph"
)

// RenderOptions contains configuration for rendering
type RenderOptions struct {
	Format        string // "svg" (only SVG is supported)
	Direction     string // "TB", "LR", "BT", "RL"
	IncludeLabels bool
	Title         string
	UseIcons      bool // Enable icon rendering (if available)
}

// RenderDiagram generates a visual diagram from the resource graph.
// It respects the provided context for cancellation.
func RenderDiagram(ctx context.Context, g *graph.Graph, outputPath string, opts RenderOptions) error {
	// Use the new export system for all formats
	return ExportDiagram(ctx, g, outputPath, opts)
}
