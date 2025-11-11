package renderer

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"strings"

	"github.com/ankek/terraform-provider-cartography/internal/graph"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// PNGRenderer handles PNG generation
type PNGRenderer struct {
	img     *image.RGBA
	options RenderOptions
}

// NewPNGRenderer creates a new PNG renderer
func NewPNGRenderer(opts RenderOptions) *PNGRenderer {
	return &PNGRenderer{
		options: opts,
	}
}

// Render generates PNG from the layout
func (r *PNGRenderer) Render(layout *Layout, g *graph.Graph) ([]byte, error) {
	// Add padding
	padding := 50.0
	width := int(layout.Width + 2*padding)
	height := int(layout.Height + 2*padding)

	// Create image
	r.img = image.NewRGBA(image.Rect(0, 0, width, height))

	// Fill white background
	draw.Draw(r.img, r.img.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)

	// Add title if present
	if r.options.Title != "" {
		r.drawTitle(r.options.Title, width, int(padding))
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

	// Encode to PNG
	buf := &bytes.Buffer{}
	if err := png.Encode(buf, r.img); err != nil {
		return nil, fmt.Errorf("failed to encode PNG: %w", err)
	}

	return buf.Bytes(), nil
}

// drawTitle draws the diagram title
func (r *PNGRenderer) drawTitle(title string, width, padding int) {
	// Draw title text centered at top
	point := fixed.Point26_6{
		X: fixed.I(width / 2),
		Y: fixed.I(padding / 2),
	}

	// Use larger font for title (simulate by drawing text multiple times slightly offset)
	d := &font.Drawer{
		Dst:  r.img,
		Src:  image.NewUniform(color.Black),
		Face: basicfont.Face7x13,
		Dot:  point,
	}

	// Center text
	textWidth := d.MeasureString(title)
	d.Dot.X -= textWidth / 2

	// Draw bold effect
	for dx := 0; dx < 2; dx++ {
		for dy := 0; dy < 2; dy++ {
			d.Dot.X = point.X - textWidth/2 + fixed.I(dx)
			d.Dot.Y = point.Y + fixed.I(dy)
			d.DrawString(title)
		}
	}
}

// renderNode renders a node
func (r *PNGRenderer) renderNode(node *NodeLayout, padding float64) {
	x := int(node.Position.X + padding)
	y := int(node.Position.Y + padding)
	w := int(node.Width)
	h := int(node.Height)

	// Get color
	col := parseColor(getNodeColor(node.Node))

	// Draw rounded rectangle
	r.drawRoundedRect(x, y, w, h, 8, col, color.RGBA{51, 51, 51, 255})

	// Draw label
	if r.options.IncludeLabels {
		centerY := y + h/2
		r.drawNodeLabel(node.Node, x+w/2, centerY)
	}
}

// renderEdge renders an edge between nodes
func (r *PNGRenderer) renderEdge(edge *EdgeLayout, padding float64) {
	if len(edge.Points) < 2 {
		return
	}

	edgeColor := color.RGBA{85, 85, 85, 255}

	// Draw line segments
	for i := 0; i < len(edge.Points)-1; i++ {
		x1 := int(edge.Points[i].X + padding)
		y1 := int(edge.Points[i].Y + padding)
		x2 := int(edge.Points[i+1].X + padding)
		y2 := int(edge.Points[i+1].Y + padding)

		r.drawLine(x1, y1, x2, y2, edgeColor, 2)
	}

	// Draw arrowhead at end
	lastIdx := len(edge.Points) - 1
	r.drawArrowhead(
		int(edge.Points[lastIdx-1].X+padding),
		int(edge.Points[lastIdx-1].Y+padding),
		int(edge.Points[lastIdx].X+padding),
		int(edge.Points[lastIdx].Y+padding),
		edgeColor,
	)

	// Draw edge label if present
	if r.options.IncludeLabels {
		label := formatEdgeLabel(edge.Edge)
		if label != "" {
			midIdx := len(edge.Points) / 2
			midX := int(edge.Points[midIdx].X + padding)
			midY := int(edge.Points[midIdx].Y + padding)
			r.drawText(label, midX, midY-5, color.RGBA{51, 51, 51, 255})
		}
	}
}

// drawNodeLabel draws the node label text
func (r *PNGRenderer) drawNodeLabel(node *graph.Node, centerX, centerY int) {
	// Node name
	name := truncate(node.Name, 20)
	r.drawText(name, centerX, centerY-10, color.White)

	// Resource type
	typeName := getResourceTypeName(node.Type)
	typeName = truncate(typeName, 25)
	r.drawText(typeName, centerX, centerY+5, color.RGBA{200, 200, 200, 255})
}

// drawRoundedRect draws a rounded rectangle
func (r *PNGRenderer) drawRoundedRect(x, y, w, h, radius int, fillColor, strokeColor color.Color) {
	// Fill
	for dy := 0; dy < h; dy++ {
		for dx := 0; dx < w; dx++ {
			px := x + dx
			py := y + dy

			// Check if within rounded corners
			inCorner := false
			if dx < radius && dy < radius {
				// Top-left corner
				if (dx-radius)*(dx-radius)+(dy-radius)*(dy-radius) > radius*radius {
					inCorner = true
				}
			} else if dx >= w-radius && dy < radius {
				// Top-right corner
				if (dx-(w-radius))*(dx-(w-radius))+(dy-radius)*(dy-radius) > radius*radius {
					inCorner = true
				}
			} else if dx < radius && dy >= h-radius {
				// Bottom-left corner
				if (dx-radius)*(dx-radius)+(dy-(h-radius))*(dy-(h-radius)) > radius*radius {
					inCorner = true
				}
			} else if dx >= w-radius && dy >= h-radius {
				// Bottom-right corner
				if (dx-(w-radius))*(dx-(w-radius))+(dy-(h-radius))*(dy-(h-radius)) > radius*radius {
					inCorner = true
				}
			}

			if !inCorner && px >= 0 && px < r.img.Bounds().Dx() && py >= 0 && py < r.img.Bounds().Dy() {
				r.img.Set(px, py, fillColor)
			}
		}
	}

	// Stroke (simplified - just draw rectangles on edges)
	for i := 0; i < 2; i++ {
		// Top and bottom
		for dx := radius; dx < w-radius; dx++ {
			r.img.Set(x+dx, y+i, strokeColor)
			r.img.Set(x+dx, y+h-1-i, strokeColor)
		}
		// Left and right
		for dy := radius; dy < h-radius; dy++ {
			r.img.Set(x+i, y+dy, strokeColor)
			r.img.Set(x+w-1-i, y+dy, strokeColor)
		}
	}
}

// drawLine draws a line between two points using Bresenham's algorithm
func (r *PNGRenderer) drawLine(x1, y1, x2, y2 int, col color.Color, thickness int) {
	dx := abs(x2 - x1)
	dy := abs(y2 - y1)
	sx := -1
	if x1 < x2 {
		sx = 1
	}
	sy := -1
	if y1 < y2 {
		sy = 1
	}
	err := dx - dy

	for {
		// Draw thick line by drawing multiple pixels
		for dt := -thickness / 2; dt <= thickness/2; dt++ {
			r.setPixel(x1+dt, y1, col)
			r.setPixel(x1, y1+dt, col)
		}

		if x1 == x2 && y1 == y2 {
			break
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x1 += sx
		}
		if e2 < dx {
			err += dx
			y1 += sy
		}
	}
}

// drawArrowhead draws an arrowhead at the end of a line
func (r *PNGRenderer) drawArrowhead(x1, y1, x2, y2 int, col color.Color) {
	// Calculate angle
	angle := math.Atan2(float64(y2-y1), float64(x2-x1))

	// Arrowhead size
	size := 10.0

	// Calculate arrowhead points
	angle1 := angle + math.Pi*0.8
	angle2 := angle - math.Pi*0.8

	px1 := x2 - int(size*math.Cos(angle1))
	py1 := y2 - int(size*math.Sin(angle1))
	px2 := x2 - int(size*math.Cos(angle2))
	py2 := y2 - int(size*math.Sin(angle2))

	// Draw arrowhead lines
	r.drawLine(x2, y2, px1, py1, col, 2)
	r.drawLine(x2, y2, px2, py2, col, 2)
}

// drawText draws text centered at the given position
func (r *PNGRenderer) drawText(text string, x, y int, col color.Color) {
	d := &font.Drawer{
		Dst:  r.img,
		Src:  image.NewUniform(col),
		Face: basicfont.Face7x13,
		Dot:  fixed.Point26_6{X: fixed.I(x), Y: fixed.I(y)},
	}

	// Center text
	textWidth := d.MeasureString(text)
	d.Dot.X -= textWidth / 2

	d.DrawString(text)
}

// setPixel sets a pixel with bounds checking
func (r *PNGRenderer) setPixel(x, y int, col color.Color) {
	if x >= 0 && x < r.img.Bounds().Dx() && y >= 0 && y < r.img.Bounds().Dy() {
		r.img.Set(x, y, col)
	}
}

// parseColor parses a hex color string
func parseColor(hexColor string) color.Color {
	hexColor = strings.TrimPrefix(hexColor, "#")

	var r, g, b uint8
	if len(hexColor) == 6 {
		fmt.Sscanf(hexColor, "%02x%02x%02x", &r, &g, &b)
	}

	return color.RGBA{r, g, b, 255}
}

// abs returns the absolute value
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
