// +build ignore

package main

import (
	"fmt"
	"os"

	"github.com/ankek/terraform-provider-cartography/internal/graph"
	"github.com/ankek/terraform-provider-cartography/internal/parser"
	"github.com/ankek/terraform-provider-cartography/internal/renderer"
)

func main() {
	fmt.Println("Generating test diagram with icons...\n")

	// Parse the test tfstate
	resources, err := parser.ParseStateFile("../broken/test.tfstate")
	if err != nil {
		fmt.Printf("Error parsing state: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Found %d resources\n", len(resources))

	// Build graph
	g := graph.BuildGraph(resources)
	fmt.Printf("Created graph with %d nodes and %d edges\n", len(g.Nodes), len(g.Edges))

	// Render with icons disabled (SVGs too small for graphviz)
	opts := renderer.RenderOptions{
		Format:        "png",
		Direction:     "TB",
		IncludeLabels: true,
		Title:         "My Infrastructure",
		UseIcons:      false, // Disabled - Lucide icons too small (24x24)
	}

	err = renderer.RenderDiagram(g, "../broken/infrastructure.png", opts)
	if err != nil {
		fmt.Printf("Error rendering diagram: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n✅ SUCCESS! Diagram generated at: broken/infrastructure.png")
	fmt.Println("Icons should now be visible!")
}
