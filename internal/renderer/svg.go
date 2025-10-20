package renderer

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"html"
	"strings"

	"github.com/ankek/terraform-provider-cartography/internal/graph"
)

// SVGRenderer handles SVG generation
type SVGRenderer struct {
	buf     *bytes.Buffer
	options RenderOptions
}

// NewSVGRenderer creates a new SVG renderer
func NewSVGRenderer(opts RenderOptions) *SVGRenderer {
	return &SVGRenderer{
		buf:     &bytes.Buffer{},
		options: opts,
	}
}

// Render generates SVG from the layout
func (r *SVGRenderer) Render(layout *Layout, g *graph.Graph) ([]byte, error) {
	// Add padding
	padding := 50.0
	width := layout.Width + 2*padding
	height := layout.Height + 2*padding

	// Start SVG
	r.writeHeader(width, height)

	// Add title if present
	if r.options.Title != "" {
		r.writeTitle(r.options.Title, width, padding)
	}

	// Render edges first (so they appear below nodes)
	for _, edgeLayout := range layout.Edges {
		r.renderEdge(edgeLayout, padding)
	}

	// Render nodes
	for nodeID, nodeLayout := range layout.Nodes {
		node := g.Nodes[nodeID]
		if node != nil {
			nodeLayout.Node = node
			r.renderNode(nodeLayout, padding)
		}
	}

	// Close SVG
	r.buf.WriteString("</svg>")

	return r.buf.Bytes(), nil
}

// writeHeader writes the SVG header
func (r *SVGRenderer) writeHeader(width, height float64) {
	r.buf.WriteString(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink"
     width="%.0f" height="%.0f" viewBox="0 0 %.0f %.0f">
<defs>
  <marker id="arrowhead" markerWidth="10" markerHeight="10" refX="9" refY="3" orient="auto">
    <polygon points="0 0, 10 3, 0 6" fill="#555555" />
  </marker>
</defs>
<rect width="100%%" height="100%%" fill="white"/>
`, width, height, width, height))
}

// writeTitle writes the diagram title
func (r *SVGRenderer) writeTitle(title string, width, padding float64) {
	r.buf.WriteString(fmt.Sprintf(`<text x="%.0f" y="%.0f" font-family="Arial, sans-serif" font-size="20" font-weight="bold" text-anchor="middle">%s</text>
`, width/2, padding/2, html.EscapeString(title)))
}

// renderNode renders a node
func (r *SVGRenderer) renderNode(node *NodeLayout, padding float64) {
	x := node.Position.X + padding
	y := node.Position.Y + padding

	// Try to get icon if enabled
	iconData := ""
	if r.options.UseIcons {
		iconPath, iconExists := GetIconForResource(node.Node.Provider, node.Node.Type)
		if iconExists {
			data, err := getIconData(iconPath)
			if err == nil {
				// Embed SVG as data URI
				iconData = embedIconData(data, iconPath)
			}
		}
	}

	// Render with or without icon
	if iconData != "" {
		r.renderNodeWithIcon(node, x, y, iconData)
	} else {
		r.renderNodeWithoutIcon(node, x, y)
	}
}

// embedIconData converts icon data to a data URI
func embedIconData(data []byte, path string) string {
	dataStr := string(data)

	// If it's already an SVG, we can embed it directly
	if strings.Contains(strings.ToLower(path), ".svg") {
		// Clean up SVG data
		dataStr = strings.TrimSpace(dataStr)
		// URL encode for data URI
		encoded := base64.StdEncoding.EncodeToString(data)
		return fmt.Sprintf("data:image/svg+xml;base64,%s", encoded)
	}

	// For PNG/JPEG
	ext := strings.ToLower(path)
	if strings.Contains(ext, ".png") {
		encoded := base64.StdEncoding.EncodeToString(data)
		return fmt.Sprintf("data:image/png;base64,%s", encoded)
	}
	if strings.Contains(ext, ".jpg") || strings.Contains(ext, ".jpeg") {
		encoded := base64.StdEncoding.EncodeToString(data)
		return fmt.Sprintf("data:image/jpeg;base64,%s", encoded)
	}

	return ""
}

// renderNodeWithIcon renders a node with an embedded icon
func (r *SVGRenderer) renderNodeWithIcon(node *NodeLayout, x, y float64, iconData string) {
	// Background box
	r.buf.WriteString(fmt.Sprintf(`<rect x="%.2f" y="%.2f" width="%.2f" height="%.2f"
    fill="white" stroke="#cccccc" stroke-width="2" rx="8"/>
`, x, y, node.Width, node.Height))

	// Icon (centered in upper portion)
	iconSize := 48.0
	iconX := x + (node.Width-iconSize)/2
	iconY := y + 10

	r.buf.WriteString(fmt.Sprintf(`<image x="%.2f" y="%.2f" width="%.2f" height="%.2f"
    xlink:href="%s" preserveAspectRatio="xMidYMid meet"/>
`, iconX, iconY, iconSize, iconSize, iconData))

	// Label below icon
	if r.options.IncludeLabels {
		labelY := y + iconSize + 30
		r.renderNodeLabel(node.Node, x+node.Width/2, labelY, node.Width)
	}
}

// renderNodeWithoutIcon renders a node without an icon (colored box)
func (r *SVGRenderer) renderNodeWithoutIcon(node *NodeLayout, x, y float64) {
	color := getNodeColor(node.Node)

	// Colored box with rounded corners
	r.buf.WriteString(fmt.Sprintf(`<rect x="%.2f" y="%.2f" width="%.2f" height="%.2f"
    fill="%s" stroke="#333333" stroke-width="2" rx="8"/>
`, x, y, node.Width, node.Height, color))

	// Label centered in box
	if r.options.IncludeLabels {
		centerY := y + node.Height/2
		r.renderNodeLabel(node.Node, x+node.Width/2, centerY, node.Width)
	}
}

// renderNodeLabel renders the node label text
func (r *SVGRenderer) renderNodeLabel(node *graph.Node, x, y, maxWidth float64) {
	// Node name
	name := truncate(node.Name, 25)
	r.buf.WriteString(fmt.Sprintf(`<text x="%.2f" y="%.2f" font-family="Arial, sans-serif"
    font-size="12" font-weight="bold" fill="#333333" text-anchor="middle">%s</text>
`, x, y, html.EscapeString(name)))

	// Resource type
	typeName := getResourceTypeName(node.Type)
	typeName = truncate(typeName, 30)
	r.buf.WriteString(fmt.Sprintf(`<text x="%.2f" y="%.2f" font-family="Arial, sans-serif"
    font-size="10" fill="#666666" text-anchor="middle">%s</text>
`, x, y+15, html.EscapeString(typeName)))
}

// renderEdge renders an edge between nodes
func (r *SVGRenderer) renderEdge(edge *EdgeLayout, padding float64) {
	if len(edge.Points) < 2 {
		return
	}

	// Build path
	pathData := fmt.Sprintf("M %.2f,%.2f",
		edge.Points[0].X+padding,
		edge.Points[0].Y+padding)

	for i := 1; i < len(edge.Points); i++ {
		pathData += fmt.Sprintf(" L %.2f,%.2f",
			edge.Points[i].X+padding,
			edge.Points[i].Y+padding)
	}

	// Draw path
	r.buf.WriteString(fmt.Sprintf(`<path d="%s" stroke="#555555" stroke-width="2"
    fill="none" marker-end="url(#arrowhead)"/>
`, pathData))

	// Add edge label if present
	if r.options.IncludeLabels {
		label := formatEdgeLabel(edge.Edge)
		if label != "" {
			// Position label at midpoint
			midIdx := len(edge.Points) / 2
			midPoint := edge.Points[midIdx]

			r.buf.WriteString(fmt.Sprintf(`<text x="%.2f" y="%.2f" font-family="Arial, sans-serif"
    font-size="9" fill="#333333" text-anchor="middle"
    style="background: white; padding: 2px;">%s</text>
`, midPoint.X+padding, midPoint.Y+padding-5, html.EscapeString(label)))
		}
	}
}
