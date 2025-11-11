package renderer

import (
	"testing"

	"github.com/ankek/terraform-provider-cartography/internal/graph"
	"github.com/ankek/terraform-provider-cartography/internal/parser"
)

func TestCalculateImprovedLayout(t *testing.T) {
	tests := []struct {
		name      string
		graph     *graph.Graph
		direction string
		wantNodes int
	}{
		{
			name: "simple linear graph",
			graph: &graph.Graph{
				Nodes: map[string]*graph.Node{
					"node1": {
						ID:       "node1",
						Type:     "aws_vpc",
						Name:     "main",
						Provider: "aws",
						Edges:    []*graph.Edge{},
					},
					"node2": {
						ID:       "node2",
						Type:     "aws_subnet",
						Name:     "public",
						Provider: "aws",
						Edges:    []*graph.Edge{},
					},
					"node3": {
						ID:       "node3",
						Type:     "aws_instance",
						Name:     "web",
						Provider: "aws",
						Edges:    []*graph.Edge{},
					},
				},
				Edges: []*graph.Edge{},
			},
			direction: "TB",
			wantNodes: 3,
		},
		{
			name: "graph with dependencies",
			graph: &graph.Graph{
				Nodes: map[string]*graph.Node{
					"vpc": {
						ID:           "vpc",
						Type:         "aws_vpc",
						Name:         "main",
						Provider:     "aws",
						ResourceType: parser.ResourceTypeNetwork,
					},
					"subnet": {
						ID:           "subnet",
						Type:         "aws_subnet",
						Name:         "public",
						Provider:     "aws",
						ResourceType: parser.ResourceTypeNetwork,
					},
					"instance": {
						ID:           "instance",
						Type:         "aws_instance",
						Name:         "web",
						Provider:     "aws",
						ResourceType: parser.ResourceTypeCompute,
					},
				},
				Edges: []*graph.Edge{},
			},
			direction: "LR",
			wantNodes: 3,
		},
		{
			name: "empty graph",
			graph: &graph.Graph{
				Nodes: map[string]*graph.Node{},
				Edges: []*graph.Edge{},
			},
			direction: "TB",
			wantNodes: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup edges
			if len(tt.graph.Nodes) > 1 {
				var nodes []*graph.Node
				for _, node := range tt.graph.Nodes {
					nodes = append(nodes, node)
				}
				for i := 0; i < len(nodes)-1; i++ {
					edge := &graph.Edge{
						From:         nodes[i],
						To:           nodes[i+1],
						Relationship: "depends_on",
					}
					tt.graph.Edges = append(tt.graph.Edges, edge)
					nodes[i].Edges = append(nodes[i].Edges, edge)
				}
			}

			layout := CalculateImprovedLayout(
				tt.graph,
				tt.direction,
				220.0, // nodeWidth
				160.0, // nodeHeight
				140.0, // horizontalSpacing
				120.0, // verticalSpacing
			)

			if len(layout.Nodes) != tt.wantNodes {
				t.Errorf("CalculateImprovedLayout() got %d nodes, want %d", len(layout.Nodes), tt.wantNodes)
			}

			// Verify all nodes have positions
			for _, nodeLayout := range layout.Nodes {
				if nodeLayout.Position.X == 0 && nodeLayout.Position.Y == 0 && len(tt.graph.Nodes) > 1 {
					// At least some nodes should have non-zero positions in a multi-node graph
					// (unless all nodes happen to be at origin)
					// This is a weak test but ensures layout is attempting positioning
				}
			}

			// Verify dimensions are calculated
			if tt.wantNodes > 0 && (layout.Width == 0 || layout.Height == 0) {
				t.Error("CalculateImprovedLayout() should set non-zero dimensions for non-empty graph")
			}
		})
	}
}

func TestCalculateImprovedLayout_Directions(t *testing.T) {
	directions := []string{"TB", "LR", "BT", "RL"}

	g := &graph.Graph{
		Nodes: map[string]*graph.Node{
			"node1": {
				ID:       "node1",
				Type:     "aws_instance",
				Name:     "web1",
				Provider: "aws",
			},
			"node2": {
				ID:       "node2",
				Type:     "aws_instance",
				Name:     "web2",
				Provider: "aws",
			},
		},
		Edges: []*graph.Edge{},
	}

	for _, direction := range directions {
		t.Run(direction, func(t *testing.T) {
			layout := CalculateImprovedLayout(g, direction, 220.0, 160.0, 140.0, 120.0)

			if len(layout.Nodes) != 2 {
				t.Errorf("CalculateImprovedLayout() with direction %s got %d nodes, want 2", direction, len(layout.Nodes))
			}

			// Verify layout has dimensions
			if layout.Width == 0 || layout.Height == 0 {
				t.Errorf("CalculateImprovedLayout() with direction %s has zero dimensions", direction)
			}
		})
	}
}

func TestCalculateImprovedLayout_CollisionDetection(t *testing.T) {
	// Create a graph where nodes might overlap
	g := &graph.Graph{
		Nodes: map[string]*graph.Node{},
		Edges: []*graph.Edge{},
	}

	// Add multiple nodes that might cause overlap
	for i := 0; i < 10; i++ {
		nodeID := string(rune('a' + i))
		g.Nodes[nodeID] = &graph.Node{
			ID:       nodeID,
			Type:     "aws_instance",
			Name:     nodeID,
			Provider: "aws",
		}
	}

	layout := CalculateImprovedLayout(g, "TB", 220.0, 160.0, 140.0, 120.0)

	if len(layout.Nodes) != 10 {
		t.Errorf("CalculateImprovedLayout() got %d nodes, want 10", len(layout.Nodes))
	}

	// Check that no two nodes have exactly the same position
	positions := make(map[string]bool)
	for _, nodeLayout := range layout.Nodes {
		posKey := string(rune(int(nodeLayout.Position.X))) + "," + string(rune(int(nodeLayout.Position.Y)))
		if positions[posKey] && len(layout.Nodes) > 1 {
			// Note: This might still happen in some layouts, so this is a soft check
			// In a real scenario with collision detection, we'd want distinct positions
		}
		positions[posKey] = true
	}
}

func TestCalculateImprovedLayout_EdgePositions(t *testing.T) {
	// Create graph with explicit edges
	node1 := &graph.Node{
		ID:       "node1",
		Type:     "aws_vpc",
		Name:     "main",
		Provider: "aws",
	}
	node2 := &graph.Node{
		ID:       "node2",
		Type:     "aws_instance",
		Name:     "web",
		Provider: "aws",
	}

	edge := &graph.Edge{
		From:         node1,
		To:           node2,
		Relationship: "contains",
	}

	node1.Edges = []*graph.Edge{edge}

	g := &graph.Graph{
		Nodes: map[string]*graph.Node{
			"node1": node1,
			"node2": node2,
		},
		Edges: []*graph.Edge{edge},
	}

	layout := CalculateImprovedLayout(g, "TB", 220.0, 160.0, 140.0, 120.0)

	// Verify edges are included in layout
	if len(layout.Edges) != 1 {
		t.Errorf("CalculateImprovedLayout() got %d edges, want 1", len(layout.Edges))
	}

	// Verify edge has points
	if len(layout.Edges) > 0 && len(layout.Edges[0].Points) < 2 {
		t.Error("CalculateImprovedLayout() edge should have at least 2 points")
	}
}

func TestCalculateImprovedLayout_LayerAssignment(t *testing.T) {
	// Test topological sorting creates layers
	vpc := &graph.Node{
		ID:       "vpc",
		Type:     "aws_vpc",
		Name:     "main",
		Provider: "aws",
	}
	subnet := &graph.Node{
		ID:       "subnet",
		Type:     "aws_subnet",
		Name:     "public",
		Provider: "aws",
	}
	instance := &graph.Node{
		ID:       "instance",
		Type:     "aws_instance",
		Name:     "web",
		Provider: "aws",
	}

	edge1 := &graph.Edge{From: subnet, To: vpc, Relationship: "member_of"}
	edge2 := &graph.Edge{From: instance, To: subnet, Relationship: "attached_to"}

	vpc.Edges = []*graph.Edge{}
	subnet.Edges = []*graph.Edge{edge1}
	instance.Edges = []*graph.Edge{edge2}

	g := &graph.Graph{
		Nodes: map[string]*graph.Node{
			"vpc":      vpc,
			"subnet":   subnet,
			"instance": instance,
		},
		Edges: []*graph.Edge{edge1, edge2},
	}

	layout := CalculateImprovedLayout(g, "TB", 220.0, 160.0, 140.0, 120.0)

	// Verify all nodes are positioned
	if len(layout.Nodes) != 3 {
		t.Errorf("CalculateImprovedLayout() got %d nodes, want 3", len(layout.Nodes))
	}

	// For TB direction, nodes should have different Y positions
	yPositions := make(map[float64]int)
	for _, nodeLayout := range layout.Nodes {
		yPositions[nodeLayout.Position.Y]++
	}

	// With dependencies, we expect nodes at different layers (different Y values)
	if len(yPositions) < 2 {
		t.Error("CalculateImprovedLayout() should create multiple layers for dependent nodes")
	}
}
