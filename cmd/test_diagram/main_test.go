package main

import (
	"testing"

	"github.com/ankek/terraform-provider-cartography/internal/graph"
	"github.com/ankek/terraform-provider-cartography/internal/parser"
)

func TestPackageImports(t *testing.T) {
	// Verify that all required packages are importable
	t.Log("All imports are valid")
}

func TestGraphCreation(t *testing.T) {
	// Test that we can create a basic graph structure
	g := &graph.Graph{
		Nodes: make(map[string]*graph.Node),
		Edges: make([]*graph.Edge, 0),
	}

	if g.Nodes == nil {
		t.Error("Graph nodes should not be nil")
	}
	if g.Edges == nil {
		t.Error("Graph edges should not be nil")
	}

	// Add a test node
	node := &graph.Node{
		ID:           "test.node",
		Name:         "node",
		Type:         "digitalocean_droplet",
		Provider:     "digitalocean",
		ResourceType: parser.ResourceTypeCompute,
		Attributes:   map[string]any{},
	}
	g.Nodes[node.ID] = node

	if len(g.Nodes) != 1 {
		t.Errorf("Expected 1 node, got %d", len(g.Nodes))
	}
}
