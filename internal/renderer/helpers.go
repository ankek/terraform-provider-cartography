package renderer

import (
	"fmt"
	"strings"

	"github.com/ankek/terraform-provider-cartography/internal/graph"
	"github.com/ankek/terraform-provider-cartography/internal/parser"
)

// formatEdgeLabel creates a label for an edge
func formatEdgeLabel(edge *graph.Edge) string {
	parts := []string{edge.Relationship}

	// Add port information
	if port, ok := edge.Metadata["port"]; ok && port != "" {
		parts = append(parts, fmt.Sprintf(":%s", port))
	}
	if protocol, ok := edge.Metadata["protocol"]; ok && protocol != "" {
		parts = append(parts, protocol)
	}

	if len(parts) > 1 {
		return strings.Join(parts, " ")
	}
	return ""
}

// getNodeColor returns the color for a node based on its type
func getNodeColor(node *graph.Node) string {
	switch node.ResourceType {
	case parser.ResourceTypeNetwork:
		return "#1E88E5" // Blue
	case parser.ResourceTypeSecurity:
		return "#E53935" // Red
	case parser.ResourceTypeCompute:
		return "#43A047" // Green
	case parser.ResourceTypeLoadBalancer:
		return "#FB8C00" // Orange
	case parser.ResourceTypeStorage:
		return "#8E24AA" // Purple
	case parser.ResourceTypeDatabase:
		return "#00ACC1" // Cyan
	case parser.ResourceTypeDNS:
		return "#FDD835" // Yellow
	case parser.ResourceTypeCertificate:
		return "#7CB342" // Light Green
	case parser.ResourceTypeSecret:
		return "#5E35B1" // Deep Purple
	case parser.ResourceTypeContainer:
		return "#039BE5" // Light Blue
	case parser.ResourceTypeCDN:
		return "#F4511E" // Deep Orange
	default:
		return "#757575" // Gray
	}
}

// getResourceTypeName returns a human-readable name for a resource type
func getResourceTypeName(resourceType string) string {
	name := strings.TrimPrefix(resourceType, "azurerm_")
	name = strings.TrimPrefix(name, "aws_")
	name = strings.TrimPrefix(name, "google_")
	name = strings.TrimPrefix(name, "digitalocean_")

	name = strings.ReplaceAll(name, "_", " ")
	words := strings.Fields(name)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + word[1:]
		}
	}

	return strings.Join(words, " ")
}

// truncate truncates a string to a maximum length
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
