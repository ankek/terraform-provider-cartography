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

// writeHeader writes the SVG header with professional styling
func (r *SVGRenderer) writeHeader(width, height float64) {
	r.buf.WriteString(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink"
     width="%.0f" height="%.0f" viewBox="0 0 %.0f %.0f">
<defs>
  <!-- Gradient for background -->
  <linearGradient id="bgGradient" x1="0%%" y1="0%%" x2="0%%" y2="100%%">
    <stop offset="0%%" style="stop-color:#f8f9fa;stop-opacity:1" />
    <stop offset="100%%" style="stop-color:#e9ecef;stop-opacity:1" />
  </linearGradient>

  <!-- Shadow filter for nodes -->
  <filter id="nodeShadow" x="-50%%" y="-50%%" width="200%%" height="200%%">
    <feGaussianBlur in="SourceAlpha" stdDeviation="3"/>
    <feOffset dx="0" dy="2" result="offsetblur"/>
    <feComponentTransfer>
      <feFuncA type="linear" slope="0.2"/>
    </feComponentTransfer>
    <feMerge>
      <feMergeNode/>
      <feMergeNode in="SourceGraphic"/>
    </feMerge>
  </filter>

  <!-- Gradient for nodes -->
  <linearGradient id="nodeGradient" x1="0%%" y1="0%%" x2="0%%" y2="100%%">
    <stop offset="0%%" style="stop-color:#ffffff;stop-opacity:1" />
    <stop offset="100%%" style="stop-color:#f8f9fa;stop-opacity:1" />
  </linearGradient>

  <!-- Narrow, sleek arrowhead -->
  <marker id="arrowhead" markerWidth="8" markerHeight="8" refX="7" refY="4" orient="auto">
    <path d="M1,1 L1,7 L7,4 z" fill="#495057" stroke="#495057" stroke-width="0.5" stroke-linejoin="miter"/>
  </marker>

  <!-- Narrow arrowhead with white outline for better visibility -->
  <marker id="arrowhead-outlined" markerWidth="8" markerHeight="8" refX="7" refY="4" orient="auto">
    <path d="M1,1 L1,7 L7,4 z" fill="#495057" stroke="white" stroke-width="0.8" stroke-linejoin="miter"/>
  </marker>

  <!-- Glow effect for icons -->
  <filter id="iconGlow">
    <feGaussianBlur stdDeviation="2" result="coloredBlur"/>
    <feMerge>
      <feMergeNode in="coloredBlur"/>
      <feMergeNode in="SourceGraphic"/>
    </feMerge>
  </filter>
</defs>

<!-- Background with gradient -->
<rect width="100%%" height="100%%" fill="url(#bgGradient)"/>

<!-- Grid pattern for professional look -->
<defs>
  <pattern id="grid" width="20" height="20" patternUnits="userSpaceOnUse">
    <path d="M 20 0 L 0 0 0 20" fill="none" stroke="#dee2e6" stroke-width="0.5" opacity="0.3"/>
  </pattern>
</defs>
<rect width="100%%" height="100%%" fill="url(#grid)"/>
`, width, height, width, height))
}

// writeTitle writes the diagram title with professional styling
func (r *SVGRenderer) writeTitle(title string, width, padding float64) {
	centerX := width / 2
	titleY := padding * 0.6

	// Title background box with rounded corners
	titleWidth := float64(len(title)*12 + 40)
	titleHeight := 40.0
	boxX := centerX - titleWidth/2
	boxY := titleY - 30

	r.buf.WriteString(fmt.Sprintf(`
<!-- Title section -->
<rect x="%.2f" y="%.2f" width="%.2f" height="%.2f"
      rx="8" ry="8" fill="white" opacity="0.9"
      stroke="#0066cc" stroke-width="2" filter="url(#nodeShadow)"/>
<text x="%.0f" y="%.0f"
      font-family="'Segoe UI', Arial, sans-serif"
      font-size="24" font-weight="600"
      fill="#2c3e50" text-anchor="middle">%s</text>
`, boxX, boxY, titleWidth, titleHeight, centerX, titleY, html.EscapeString(title)))
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

// renderNodeWithIcon renders a node with an embedded icon and modern styling
func (r *SVGRenderer) renderNodeWithIcon(node *NodeLayout, x, y float64, iconData string) {
	// Get accent color based on resource type
	accentColor := getAccentColor(node.Node)

	// Card-style background with gradient and shadow
	r.buf.WriteString(fmt.Sprintf(`
<!-- Node: %s -->
<g class="node">
  <!-- Card background -->
  <rect x="%.2f" y="%.2f" width="%.2f" height="%.2f"
        rx="14" ry="14"
        fill="url(#nodeGradient)"
        stroke="%s" stroke-width="3"
        filter="url(#nodeShadow)"/>

  <!-- Accent bar at top -->
  <rect x="%.2f" y="%.2f" width="%.2f" height="6"
        rx="14" ry="14"
        fill="%s" opacity="0.85"/>

  <!-- Icon (clean, no circle background) -->
  <image x="%.2f" y="%.2f" width="%.2f" height="%.2f"
         xlink:href="%s" preserveAspectRatio="xMidYMid meet"/>
`,
		node.Node.Name,
		x, y, node.Width, node.Height,
		accentColor,
		x, y, node.Width,
		accentColor,
		x+node.Width/2-32, y+60-32, 64.0, 64.0,
		iconData))

	// Label below icon
	if r.options.IncludeLabels {
		labelY := y + 115
		r.renderNodeLabel(node.Node, x+node.Width/2, labelY, node.Width)
	}

	r.buf.WriteString("</g>\n")
}

// renderNodeWithoutIcon renders a node without an icon with modern gradient styling
func (r *SVGRenderer) renderNodeWithoutIcon(node *NodeLayout, x, y float64) {
	color := getNodeColor(node.Node)
	accentColor := getAccentColor(node.Node)

	// Create a gradient ID for this node
	gradientID := fmt.Sprintf("grad_%s", strings.ReplaceAll(node.Node.ID, ".", "_"))

	// Add gradient definition
	r.buf.WriteString(fmt.Sprintf(`
<defs>
  <linearGradient id="%s" x1="0%%" y1="0%%" x2="0%%" y2="100%%">
    <stop offset="0%%" style="stop-color:%s;stop-opacity:0.9" />
    <stop offset="100%%" style="stop-color:%s;stop-opacity:1" />
  </linearGradient>
</defs>
`, gradientID, lightenColor(color, 20), color))

	// Card with gradient and shadow
	r.buf.WriteString(fmt.Sprintf(`
<g class="node">
  <rect x="%.2f" y="%.2f" width="%.2f" height="%.2f"
        rx="12" ry="12"
        fill="url(#%s)"
        stroke="%s" stroke-width="2.5"
        filter="url(#nodeShadow)"/>
`,
		x, y, node.Width, node.Height,
		gradientID,
		accentColor))

	// Label centered in box with better contrast
	if r.options.IncludeLabels {
		centerY := y + node.Height/2
		r.renderNodeLabel(node.Node, x+node.Width/2, centerY, node.Width)
	}

	r.buf.WriteString("</g>\n")
}

// renderNodeLabel renders the node label text with professional typography
func (r *SVGRenderer) renderNodeLabel(node *graph.Node, x, y, maxWidth float64) {
	// Node name with shadow for better readability
	name := truncate(node.Name, 25)
	r.buf.WriteString(fmt.Sprintf(`
  <!-- Label shadow for better readability -->
  <text x="%.2f" y="%.2f" font-family="'Segoe UI', Arial, sans-serif"
        font-size="14" font-weight="600" fill="black" opacity="0.1"
        text-anchor="middle">%s</text>
  <!-- Main label -->
  <text x="%.2f" y="%.2f" font-family="'Segoe UI', Arial, sans-serif"
        font-size="14" font-weight="600" fill="#2c3e50"
        text-anchor="middle">%s</text>
`, x+1, y+1, html.EscapeString(name), x, y, html.EscapeString(name)))

	// Resource type with subtle styling
	typeName := getResourceTypeName(node.Type)
	typeName = truncate(typeName, 30)
	r.buf.WriteString(fmt.Sprintf(`
  <text x="%.2f" y="%.2f" font-family="'Segoe UI', Arial, sans-serif"
        font-size="11" fill="#6c757d" opacity="0.9"
        text-anchor="middle">%s</text>
`, x, y+18, html.EscapeString(typeName)))
}

// renderEdge renders an edge between nodes with modern styling and curved lines
func (r *SVGRenderer) renderEdge(edge *EdgeLayout, padding float64) {
	if len(edge.Points) < 2 {
		return
	}

	// Build path - use smooth curves for multi-point paths
	var pathData string

	if len(edge.Points) == 2 {
		// Straight line for directly connected nodes
		pathData = fmt.Sprintf("M %.2f,%.2f L %.2f,%.2f",
			edge.Points[0].X+padding, edge.Points[0].Y+padding,
			edge.Points[1].X+padding, edge.Points[1].Y+padding)
	} else if len(edge.Points) == 3 {
		// Quadratic Bezier for 3-point paths (smoother curves)
		pathData = fmt.Sprintf("M %.2f,%.2f Q %.2f,%.2f %.2f,%.2f",
			edge.Points[0].X+padding, edge.Points[0].Y+padding,
			edge.Points[1].X+padding, edge.Points[1].Y+padding,
			edge.Points[2].X+padding, edge.Points[2].Y+padding)
	} else {
		// Smooth curve through multiple points using cubic Bezier
		pathData = fmt.Sprintf("M %.2f,%.2f",
			edge.Points[0].X+padding,
			edge.Points[0].Y+padding)

		// Use smooth curve through all points
		for i := 1; i < len(edge.Points)-1; i++ {
			// Calculate control point for smoother curves
			curr := edge.Points[i]
			next := edge.Points[i+1]
			cp1X := curr.X + (next.X-curr.X)*0.3
			cp1Y := curr.Y + (next.Y-curr.Y)*0.3
			cp2X := curr.X + (next.X-curr.X)*0.7
			cp2Y := curr.Y + (next.Y-curr.Y)*0.7

			pathData += fmt.Sprintf(" C %.2f,%.2f %.2f,%.2f %.2f,%.2f",
				cp1X+padding, cp1Y+padding,
				cp2X+padding, cp2Y+padding,
				next.X+padding, next.Y+padding)
		}
	}

	// Draw path with compact, professional styling
	r.buf.WriteString(fmt.Sprintf(`
<!-- Edge connection -->
<g class="edge">
  <!-- White outline for contrast against background -->
  <path d="%s" stroke="white" stroke-width="3.5" opacity="0.7"
        fill="none" stroke-linecap="round" stroke-linejoin="round"/>
  <!-- Shadow for depth -->
  <path d="%s" stroke="#000000" stroke-width="2.5" opacity="0.12"
        fill="none" stroke-linecap="round" stroke-linejoin="round"/>
  <!-- Main connection line with enhanced visibility -->
  <path d="%s" stroke="#495057" stroke-width="1.5"
        fill="none" marker-end="url(#arrowhead-outlined)"
        stroke-linecap="round" stroke-linejoin="round" opacity="0.85"/>
`, pathData, pathData, pathData))

	// Add edge label if present
	if r.options.IncludeLabels {
		label := formatEdgeLabel(edge.Edge)
		if label != "" {
			// Position label at midpoint
			midIdx := len(edge.Points) / 2
			midPoint := edge.Points[midIdx]

			// Label with background box for readability
			labelWidth := float64(len(label)*7 + 12)
			labelHeight := 22.0
			labelX := midPoint.X + padding
			labelY := midPoint.Y + padding - 5

			r.buf.WriteString(fmt.Sprintf(`
  <!-- Edge label background -->
  <rect x="%.2f" y="%.2f" width="%.2f" height="%.2f"
        rx="4" ry="4" fill="white" opacity="0.95"
        stroke="#6c757d" stroke-width="1"/>
  <!-- Edge label text -->
  <text x="%.2f" y="%.2f" font-family="'Segoe UI', Arial, sans-serif"
        font-size="10" font-weight="500" fill="#495057"
        text-anchor="middle">%s</text>
`, labelX-labelWidth/2, labelY-16, labelWidth, labelHeight,
				labelX, labelY, html.EscapeString(label)))
		}
	}

	r.buf.WriteString("</g>\n")
}
