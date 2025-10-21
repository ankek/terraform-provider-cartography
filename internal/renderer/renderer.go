package renderer

import (
	"github.com/ankek/terraform-provider-cartography/internal/graph"
)

// RenderOptions contains configuration for rendering
type RenderOptions struct {
	Format        string // "png" or "svg"
	Direction     string // "TB", "LR", "BT", "RL"
	IncludeLabels bool
	Title         string
	UseIcons      bool   // Enable icon rendering (if available)
}

// RenderDiagram generates a visual diagram from the resource graph
func RenderDiagram(g *graph.Graph, outputPath string, opts RenderOptions) error {
	// Use the new export system for all formats
	return ExportDiagram(g, outputPath, opts)
}
