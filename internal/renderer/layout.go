package renderer

import (
	"sort"

	"github.com/ankek/terraform-provider-cartography/internal/graph"
)

// Point represents a 2D coordinate
type Point struct {
	X, Y float64
}

// NodeLayout represents the layout information for a node
type NodeLayout struct {
	Node     *graph.Node
	Position Point
	Width    float64
	Height   float64
	Layer    int // Hierarchical layer (0 = top/left)
}

// EdgeLayout represents the layout information for an edge
type EdgeLayout struct {
	Edge   *graph.Edge
	Points []Point // Control points for the edge path
}

// Layout represents the complete graph layout
type Layout struct {
	Nodes     map[string]*NodeLayout
	Edges     []*EdgeLayout
	Width     float64
	Height    float64
	Direction string // TB, LR, BT, RL
}

// CalculateLayout performs hierarchical graph layout
func CalculateLayout(g *graph.Graph, direction string, nodeWidth, nodeHeight, horizontalSpacing, verticalSpacing float64) *Layout {
	layout := &Layout{
		Nodes:     make(map[string]*NodeLayout),
		Edges:     []*EdgeLayout{},
		Direction: direction,
	}

	if len(g.Nodes) == 0 {
		return layout
	}

	// Step 1: Assign layers (topological sort)
	layers := assignLayers(g)

	// Step 2: Minimize edge crossings (simple approach)
	orderNodesInLayers(layers, g)

	// Step 3: Assign coordinates
	assignCoordinates(layout, layers, direction, nodeWidth, nodeHeight, horizontalSpacing, verticalSpacing)

	// Step 4: Calculate edge paths
	calculateEdgePaths(layout, g)

	return layout
}

// assignLayers performs topological sorting to assign nodes to layers
func assignLayers(g *graph.Graph) [][]string {
	// Calculate in-degree for each node
	inDegree := make(map[string]int)
	outEdges := make(map[string][]string)

	for id := range g.Nodes {
		inDegree[id] = 0
	}

	for _, edge := range g.Edges {
		inDegree[edge.To.ID]++
		outEdges[edge.From.ID] = append(outEdges[edge.From.ID], edge.To.ID)
	}

	// Find nodes with no incoming edges (roots)
	var queue []string
	for id, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, id)
		}
	}

	// If no roots found (cycle or disconnected), pick arbitrary starting nodes
	if len(queue) == 0 {
		for id := range g.Nodes {
			queue = append(queue, id)
			break
		}
	}

	// Assign layers using BFS
	layers := [][]string{}
	layer := 0
	nodeLayer := make(map[string]int)

	for len(queue) > 0 {
		currentLayer := queue
		queue = []string{}
		layers = append(layers, []string{})

		for _, nodeID := range currentLayer {
			if _, exists := nodeLayer[nodeID]; exists {
				continue // Already assigned
			}

			nodeLayer[nodeID] = layer
			layers[layer] = append(layers[layer], nodeID)

			// Add children to next layer
			for _, childID := range outEdges[nodeID] {
				if _, exists := nodeLayer[childID]; !exists {
					queue = append(queue, childID)
				}
			}
		}

		layer++
	}

	// Handle any unassigned nodes (in disconnected components)
	for id := range g.Nodes {
		if _, exists := nodeLayer[id]; !exists {
			if len(layers) == 0 {
				layers = append(layers, []string{})
			}
			layers[len(layers)-1] = append(layers[len(layers)-1], id)
		}
	}

	return layers
}

// orderNodesInLayers orders nodes within each layer to minimize crossings
func orderNodesInLayers(layers [][]string, g *graph.Graph) {
	// Simple approach: sort by number of connections
	for i := range layers {
		layer := layers[i]
		sort.Slice(layer, func(a, b int) bool {
			nodeA := layer[a]
			nodeB := layer[b]

			// Count connections
			connectionsA := 0
			connectionsB := 0

			for _, edge := range g.Edges {
				if edge.From.ID == nodeA || edge.To.ID == nodeA {
					connectionsA++
				}
				if edge.From.ID == nodeB || edge.To.ID == nodeB {
					connectionsB++
				}
			}

			// More connected nodes go to the center
			return connectionsA > connectionsB
		})
	}
}

// assignCoordinates assigns X,Y coordinates to nodes based on layout
func assignCoordinates(layout *Layout, layers [][]string, direction string, nodeWidth, nodeHeight, hSpacing, vSpacing float64) {
	// Calculate maximum width (for centering)
	maxNodesInLayer := 0
	for _, layer := range layers {
		if len(layer) > maxNodesInLayer {
			maxNodesInLayer = len(layer)
		}
	}

	// Assign coordinates based on direction
	for layerIdx, layer := range layers {
		layerWidth := float64(len(layer)-1)*hSpacing + float64(len(layer))*nodeWidth
		startOffset := (float64(maxNodesInLayer)*nodeWidth + float64(maxNodesInLayer-1)*hSpacing - layerWidth) / 2

		for nodeIdx, nodeID := range layer {
			node := layout.Nodes[nodeID]
			if node == nil {
				node = &NodeLayout{
					Node:   nil, // Will be set later
					Width:  nodeWidth,
					Height: nodeHeight,
					Layer:  layerIdx,
				}
				layout.Nodes[nodeID] = node
			}

			var x, y float64

			switch direction {
			case "TB": // Top to Bottom
				x = startOffset + float64(nodeIdx)*(nodeWidth+hSpacing)
				y = float64(layerIdx) * (nodeHeight + vSpacing)
			case "BT": // Bottom to Top
				x = startOffset + float64(nodeIdx)*(nodeWidth+hSpacing)
				y = float64(len(layers)-1-layerIdx) * (nodeHeight + vSpacing)
			case "LR": // Left to Right
				x = float64(layerIdx) * (nodeWidth + hSpacing)
				y = startOffset + float64(nodeIdx)*(nodeHeight+vSpacing)
			case "RL": // Right to Left
				x = float64(len(layers)-1-layerIdx) * (nodeWidth + hSpacing)
				y = startOffset + float64(nodeIdx)*(nodeHeight+vSpacing)
			default: // Default to TB
				x = startOffset + float64(nodeIdx)*(nodeWidth+hSpacing)
				y = float64(layerIdx) * (nodeHeight + vSpacing)
			}

			node.Position = Point{X: x, Y: y}
		}
	}

	// Calculate total dimensions
	maxX := 0.0
	maxY := 0.0
	for _, node := range layout.Nodes {
		if node.Position.X+node.Width > maxX {
			maxX = node.Position.X + node.Width
		}
		if node.Position.Y+node.Height > maxY {
			maxY = node.Position.Y + node.Height
		}
	}

	layout.Width = maxX + hSpacing
	layout.Height = maxY + vSpacing
}

// calculateEdgePaths calculates paths for edges
func calculateEdgePaths(layout *Layout, g *graph.Graph) {
	for _, edge := range g.Edges {
		fromNode := layout.Nodes[edge.From.ID]
		toNode := layout.Nodes[edge.To.ID]

		if fromNode == nil || toNode == nil {
			continue
		}

		edgeLayout := &EdgeLayout{
			Edge:   edge,
			Points: calculateEdgePoints(fromNode, toNode, layout.Direction),
		}

		layout.Edges = append(layout.Edges, edgeLayout)
	}
}

// calculateEdgePoints calculates the path points for an edge
func calculateEdgePoints(from, to *NodeLayout, direction string) []Point {
	// Calculate connection points based on direction
	var startPoint, endPoint Point

	switch direction {
	case "TB": // Top to Bottom
		startPoint = Point{
			X: from.Position.X + from.Width/2,
			Y: from.Position.Y + from.Height,
		}
		endPoint = Point{
			X: to.Position.X + to.Width/2,
			Y: to.Position.Y,
		}
	case "BT": // Bottom to Top
		startPoint = Point{
			X: from.Position.X + from.Width/2,
			Y: from.Position.Y,
		}
		endPoint = Point{
			X: to.Position.X + to.Width/2,
			Y: to.Position.Y + to.Height,
		}
	case "LR": // Left to Right
		startPoint = Point{
			X: from.Position.X + from.Width,
			Y: from.Position.Y + from.Height/2,
		}
		endPoint = Point{
			X: to.Position.X,
			Y: to.Position.Y + to.Height/2,
		}
	case "RL": // Right to Left
		startPoint = Point{
			X: from.Position.X,
			Y: from.Position.Y + from.Height/2,
		}
		endPoint = Point{
			X: to.Position.X + to.Width,
			Y: to.Position.Y + to.Height/2,
		}
	default: // Default TB
		startPoint = Point{
			X: from.Position.X + from.Width/2,
			Y: from.Position.Y + from.Height,
		}
		endPoint = Point{
			X: to.Position.X + to.Width/2,
			Y: to.Position.Y,
		}
	}

	// Simple straight line or orthogonal routing
	return []Point{startPoint, endPoint}
}
