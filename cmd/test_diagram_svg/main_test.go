package main

import (
	"testing"

	"github.com/ankek/terraform-provider-cartography/internal/graph"
	"github.com/ankek/terraform-provider-cartography/internal/parser"
	"github.com/ankek/terraform-provider-cartography/internal/renderer"
)

func TestPackageImports(t *testing.T) {
	// Verify that all required packages are importable
	t.Log("All imports are valid")
}

func TestRenderOptions(t *testing.T) {
	// Test that we can create render options
	opts := renderer.RenderOptions{
		Format:        "svg",
		Direction:     "TB",
		IncludeLabels: true,
		Title:         "Test Infrastructure",
		UseIcons:      true,
	}

	if opts.Format != "svg" {
		t.Errorf("Expected format 'svg', got '%s'", opts.Format)
	}
	if opts.Direction != "TB" {
		t.Errorf("Expected direction 'TB', got '%s'", opts.Direction)
	}
	if !opts.IncludeLabels {
		t.Error("Expected IncludeLabels to be true")
	}
	if !opts.UseIcons {
		t.Error("Expected UseIcons to be true")
	}
}

func TestGraphWithEdges(t *testing.T) {
	// Test that we can create a graph with edges
	g := &graph.Graph{
		Nodes: make(map[string]*graph.Node),
		Edges: make([]*graph.Edge, 0),
	}

	node1 := &graph.Node{
		ID:           "digitalocean_droplet.web",
		Name:         "web",
		Type:         "digitalocean_droplet",
		Provider:     "digitalocean",
		ResourceType: parser.ResourceTypeCompute,
		Attributes:   map[string]any{},
	}
	g.Nodes[node1.ID] = node1

	node2 := &graph.Node{
		ID:           "digitalocean_firewall.web-fw",
		Name:         "web-fw",
		Type:         "digitalocean_firewall",
		Provider:     "digitalocean",
		ResourceType: parser.ResourceTypeSecurity,
		Attributes:   map[string]any{},
	}
	g.Nodes[node2.ID] = node2

	edge := &graph.Edge{
		From:         node2,
		To:           node1,
		Relationship: "protects",
		Metadata: map[string]string{
			"port":     "22",
			"protocol": "tcp",
		},
	}
	g.Edges = append(g.Edges, edge)

	if len(g.Nodes) != 2 {
		t.Errorf("Expected 2 nodes, got %d", len(g.Nodes))
	}
	if len(g.Edges) != 1 {
		t.Errorf("Expected 1 edge, got %d", len(g.Edges))
	}
}
