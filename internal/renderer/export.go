package renderer

import (
	"context"
	"fmt"
	"strings"

	"github.com/ankek/terraform-provider-cartography/internal/graph"
)

// ExportDiagram exports a diagram in SVG format with context support
func ExportDiagram(ctx context.Context, g *graph.Graph, outputPath string, opts RenderOptions) error {
	format := strings.ToLower(opts.Format)

	// Check context before starting
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Only SVG format is supported
	if format != "svg" {
		return fmt.Errorf("unsupported format: %s (only SVG is supported)", format)
	}

	// Calculate layout with improved algorithm (prevents overlaps, adds curves)
	nodeWidth := 220.0   // Slightly wider for better visibility
	nodeHeight := 160.0  // Taller for better icon display
	horizontalSpacing := 140.0  // More space between nodes
	verticalSpacing := 120.0    // More vertical space

	layout := CalculateImprovedLayout(g, opts.Direction, nodeWidth, nodeHeight, horizontalSpacing, verticalSpacing)

	// Generate SVG
	svgRenderer := NewSVGRenderer(opts)
	svgData, err := svgRenderer.Render(layout, g)
	if err != nil {
		return fmt.Errorf("failed to generate SVG: %w", err)
	}

	return writeFile(outputPath, svgData)
}
