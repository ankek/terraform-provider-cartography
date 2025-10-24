package main

import (
	"context"
	"fmt"
	"github.com/ankek/terraform-provider-cartography/internal/graph"
	"github.com/ankek/terraform-provider-cartography/internal/parser"
	"github.com/ankek/terraform-provider-cartography/internal/renderer"
)

func main() {
	fmt.Println("Testing icon rendering with temporary files...")

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

	// Add DigitalOcean Monitor Alert
	alert := &graph.Node{
		ID:       "digitalocean_monitor_alert.cpu_alert",
		Name:     "cpu_alert",
		Type:     "digitalocean_monitor_alert",
		Provider: "digitalocean",
		ResourceType: parser.ResourceTypeSecret,
		Attributes: map[string]any{},
	}
	g.Nodes[alert.ID] = alert

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

	g.Edges = append(g.Edges, &graph.Edge{
		From: alert,
		To:   droplet,
		Relationship: "monitors",
		Metadata: map[string]string{},
	})

	// Test rendering with icons
	opts := renderer.RenderOptions{
		Format:        "svg",
		Direction:     "TB",
		IncludeLabels: true,
		Title:         "My Infrastructure",
		UseIcons:      true, // ENABLE ICONS
	}

	fmt.Println("Rendering diagram with icons enabled...")
	ctx := context.Background()
	err := renderer.RenderDiagram(ctx, g, "broken/infrastructure.svg", opts)

	if err != nil {
		fmt.Printf("❌ FAIL: %v\n", err)
		return
	}

	fmt.Println("✅ SUCCESS: Diagram rendered with icons!")
	fmt.Println("\nOutput: broken/infrastructure.svg")
	fmt.Println("\nOpen the SVG in your browser to see the beautiful diagram!")
}
