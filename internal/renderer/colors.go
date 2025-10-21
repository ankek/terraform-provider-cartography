package renderer

import (
	"fmt"
	"strconv"

	"github.com/ankek/terraform-provider-cartography/internal/graph"
	"github.com/ankek/terraform-provider-cartography/internal/parser"
)

// getAccentColor returns a modern accent color based on resource type
func getAccentColor(node *graph.Node) string {
	switch node.ResourceType {
	case parser.ResourceTypeNetwork:
		return "#2196F3" // Modern Blue
	case parser.ResourceTypeSecurity:
		return "#F44336" // Material Red
	case parser.ResourceTypeCompute:
		return "#4CAF50" // Material Green
	case parser.ResourceTypeLoadBalancer:
		return "#FF9800" // Material Orange
	case parser.ResourceTypeStorage:
		return "#9C27B0" // Material Purple
	case parser.ResourceTypeDatabase:
		return "#00BCD4" // Material Cyan
	case parser.ResourceTypeDNS:
		return "#FFC107" // Material Amber
	case parser.ResourceTypeCertificate:
		return "#8BC34A" // Material Light Green
	case parser.ResourceTypeSecret:
		return "#673AB7" // Material Deep Purple
	case parser.ResourceTypeContainer:
		return "#03A9F4" // Material Light Blue
	case parser.ResourceTypeCDN:
		return "#FF5722" // Material Deep Orange
	default:
		return "#607D8B" // Material Blue Grey
	}
}

// lightenColor lightens a hex color by a percentage
func lightenColor(hexColor string, percent int) string {
	// Parse hex color
	if hexColor[0] == '#' {
		hexColor = hexColor[1:]
	}

	// Convert to RGB
	r, _ := strconv.ParseInt(hexColor[0:2], 16, 64)
	g, _ := strconv.ParseInt(hexColor[2:4], 16, 64)
	b, _ := strconv.ParseInt(hexColor[4:6], 16, 64)

	// Lighten
	factor := float64(percent) / 100.0
	r = int64(float64(r) + (255-float64(r))*factor)
	g = int64(float64(g) + (255-float64(g))*factor)
	b = int64(float64(b) + (255-float64(b))*factor)

	// Clamp values
	if r > 255 {
		r = 255
	}
	if g > 255 {
		g = 255
	}
	if b > 255 {
		b = 255
	}

	return fmt.Sprintf("#%02X%02X%02X", r, g, b)
}

// darkenColor darkens a hex color by a percentage
func darkenColor(hexColor string, percent int) string {
	// Parse hex color
	if hexColor[0] == '#' {
		hexColor = hexColor[1:]
	}

	// Convert to RGB
	r, _ := strconv.ParseInt(hexColor[0:2], 16, 64)
	g, _ := strconv.ParseInt(hexColor[2:4], 16, 64)
	b, _ := strconv.ParseInt(hexColor[4:6], 16, 64)

	// Darken
	factor := 1.0 - (float64(percent) / 100.0)
	r = int64(float64(r) * factor)
	g = int64(float64(g) * factor)
	b = int64(float64(b) * factor)

	// Clamp values
	if r < 0 {
		r = 0
	}
	if g < 0 {
		g = 0
	}
	if b < 0 {
		b = 0
	}

	return fmt.Sprintf("#%02X%02X%02X", r, g, b)
}
