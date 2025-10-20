package renderer

import (
	"fmt"
	"os"
	"strings"

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
	// Node dimensions and spacing
	nodeWidth := 180.0
	nodeHeight := 100.0
	horizontalSpacing := 100.0
	verticalSpacing := 80.0

	// Calculate layout
	layout := CalculateLayout(g, opts.Direction, nodeWidth, nodeHeight, horizontalSpacing, verticalSpacing)

	// Render based on format
	var data []byte
	var err error

	format := strings.ToLower(opts.Format)
	switch format {
	case "svg":
		renderer := NewSVGRenderer(opts)
		data, err = renderer.Render(layout, g)
	case "png":
		renderer := NewPNGRenderer(opts)
		data, err = renderer.Render(layout, g)
	default:
		// Default to SVG for better quality and no external dependencies
		renderer := NewSVGRenderer(opts)
		data, err = renderer.Render(layout, g)
	}

	if err != nil {
		return fmt.Errorf("failed to render diagram: %w", err)
	}

	// Write to file
	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	return nil
}
