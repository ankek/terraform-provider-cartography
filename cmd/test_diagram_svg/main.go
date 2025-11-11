package main

import (
	"context"
	"fmt"
	"github.com/ankek/terraform-provider-cartography/internal/graph"
	"github.com/ankek/terraform-provider-cartography/internal/parser"
	"github.com/ankek/terraform-provider-cartography/internal/renderer"
)

func main() {
	fmt.Println("Testing icon rendering with SVG output...")

	// Create test graph with DigitalOcean resources
	g := &graph.Graph{
		Nodes: make(map[string]*graph.Node),
		Edges: make([]*graph.Edge, 0),
	}

	// Add DigitalOcean Droplet
	droplet := &graph.Node{
		ID:       "digitalocean_droplet.web-1",
		Name:     "web-1",
		Type:     "digitalocean_droplet",
		Provider: "digitalocean",
		ResourceType: parser.ResourceTypeCompute,
		Attributes: map[string]any{
			"region": "ams3",
		},
	}
	g.Nodes[droplet.ID] = droplet

	// Add DigitalOcean Firewall
	firewall := &graph.Node{
		ID:       "digitalocean_firewall.web-1-fw-rules",
		Name:     "web-1-fw-rules",
		Type:     "digitalocean_firewall",
		Provider: "digitalocean",
		ResourceType: parser.ResourceTypeSecurity,
		Attributes: map[string]any{},
	}
	g.Nodes[firewall.ID] = firewall

	// Add DigitalOcean SSH Key
	sshkey := &graph.Node{
		ID:       "digitalocean_ssh_key.terraform_two",
		Name:     "terraform_two",
		Type:     "digitalocean_ssh_key",
		Provider: "digitalocean",
		ResourceType: parser.ResourceTypeSecret,
		Attributes: map[string]any{},
	}
	g.Nodes[sshkey.ID] = sshkey

	// Add edges
	g.Edges = append(g.Edges, &graph.Edge{
		From: firewall,
		To:   droplet,
		Relationship: "protects :22 tcp",
		Metadata: map[string]string{
			"port": "22",
			"protocol": "tcp",
		},
	})

	// Test rendering with icons - SVG OUTPUT
	opts := renderer.RenderOptions{
		Format:        "svg",
		Direction:     "TB",
		IncludeLabels: true,
		Title:         "My Infrastructure",
		UseIcons:      true, // ENABLE ICONS
	}

	// Check icon availability before rendering
	fmt.Println("Checking icon availability...")
	for id, node := range g.Nodes {
		iconPath, exists := renderer.GetIconForResource(node.Provider, node.Type)
		fmt.Printf("  %s: icon_path=%s, exists=%v\n", id, iconPath, exists)
	}

	fmt.Println("\nRendering diagram with icons enabled (SVG output)...")
	ctx := context.Background()
	err := renderer.RenderDiagram(ctx, g, "broken/infrastructure_test.svg", opts)

	if err != nil {
		fmt.Printf("❌ FAIL: %v\n", err)
		return
	}

	fmt.Println("✅ SUCCESS: Diagram rendered with icons!")
	fmt.Println("\nOutput: broken/infrastructure_test.svg")
}
