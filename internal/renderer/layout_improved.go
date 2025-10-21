package renderer

import (
	"math"
	"sort"

	"github.com/ankek/terraform-provider-cartography/internal/graph"
	"github.com/ankek/terraform-provider-cartography/internal/parser"
)

// ImprovedLayout creates a layout with better spacing and no overlaps
type ImprovedLayout struct {
	*Layout
	nodesByLayer map[int][]*NodeLayout
	groupings    map[parser.ResourceType][]*NodeLayout
}

// CalculateImprovedLayout creates a professional layout with proper spacing
func CalculateImprovedLayout(g *graph.Graph, direction string, nodeWidth, nodeHeight, hSpacing, vSpacing float64) *Layout {
	// Increase spacing for better visibility
	enhancedHSpacing := hSpacing * 1.5  // 180px between nodes horizontally
	enhancedVSpacing := vSpacing * 1.5  // 150px between nodes vertically

	layout := &Layout{
		Nodes:     make(map[string]*NodeLayout),
		Edges:     []*EdgeLayout{},
		Direction: direction,
	}

	if len(g.Nodes) == 0 {
		return layout
	}

	improved := &ImprovedLayout{
		Layout:       layout,
		nodesByLayer: make(map[int][]*NodeLayout),
		groupings:    make(map[parser.ResourceType][]*NodeLayout),
	}

	// Step 1: Assign layers with better distribution
	layers := improved.assignLayersWithGrouping(g)

	// Step 2: Minimize crossings using barycenter heuristic
	improved.minimizeCrossings(layers, g)

	// Step 3: Assign coordinates with collision avoidance
	improved.assignCoordinatesWithSpacing(layers, direction, nodeWidth, nodeHeight, enhancedHSpacing, enhancedVSpacing)

	// Step 4: Detect and resolve overlaps
	improved.resolveOverlaps(nodeWidth, nodeHeight)

	// Step 5: Route edges intelligently to avoid overlaps
	improved.routeEdgesWithAvoidance(g, nodeWidth, nodeHeight)

	return layout
}

// routeEdgesWithAvoidance uses the edge router to prevent line overlaps
func (il *ImprovedLayout) routeEdgesWithAvoidance(g *graph.Graph, nodeWidth, nodeHeight float64) {
	router := NewEdgeRouter(il.Layout, nodeWidth, nodeHeight)
	il.Edges = router.RouteEdges(g)
}

// assignLayersWithGrouping assigns layers while grouping related resources
func (il *ImprovedLayout) assignLayersWithGrouping(g *graph.Graph) [][]string {
	// Calculate in-degree and out-edges
	inDegree := make(map[string]int)
	outEdges := make(map[string][]string)
	inEdges := make(map[string][]string)

	for id := range g.Nodes {
		inDegree[id] = 0
	}

	for _, edge := range g.Edges {
		inDegree[edge.To.ID]++
		outEdges[edge.From.ID] = append(outEdges[edge.From.ID], edge.To.ID)
		inEdges[edge.To.ID] = append(inEdges[edge.To.ID], edge.From.ID)
	}

	// Modified BFS that considers resource types
	layers := [][]string{}
	nodeLayer := make(map[string]int)
	processed := make(map[string]bool)

	// Start with roots (no incoming edges)
	var currentLayer []string
	for id, deg := range inDegree {
		if deg == 0 {
			currentLayer = append(currentLayer, id)
		}
	}

	// If no roots (cycles), start with security/network resources
	if len(currentLayer) == 0 {
		for id, node := range g.Nodes {
			if node.ResourceType == parser.ResourceTypeSecurity ||
				node.ResourceType == parser.ResourceTypeNetwork {
				currentLayer = append(currentLayer, id)
				if len(currentLayer) >= 3 {
					break
				}
			}
		}
		// If still empty, just pick any
		if len(currentLayer) == 0 {
			for id := range g.Nodes {
				currentLayer = append(currentLayer, id)
				break
			}
		}
	}

	layerIdx := 0
	for len(processed) < len(g.Nodes) && layerIdx < 20 {
		if len(currentLayer) == 0 {
			// Find unprocessed nodes
			for id := range g.Nodes {
				if !processed[id] {
					currentLayer = append(currentLayer, id)
					break
				}
			}
		}

		// Group current layer by resource type for better visualization
		groupedLayer := il.groupByResourceType(currentLayer, g)
		layers = append(layers, groupedLayer)

		for _, id := range groupedLayer {
			nodeLayer[id] = layerIdx
			processed[id] = true
		}

		// Prepare next layer
		nextLayer := []string{}
		seen := make(map[string]bool)

		for _, id := range currentLayer {
			for _, childID := range outEdges[id] {
				if !processed[childID] && !seen[childID] {
					// Check if all parents are processed
					allParentsProcessed := true
					for _, parentID := range inEdges[childID] {
						if !processed[parentID] {
							allParentsProcessed = false
							break
						}
					}

					if allParentsProcessed {
						nextLayer = append(nextLayer, childID)
						seen[childID] = true
					}
				}
			}
		}

		currentLayer = nextLayer
		layerIdx++
	}

	return layers
}

// groupByResourceType groups nodes by their resource type for better layout
func (il *ImprovedLayout) groupByResourceType(nodeIDs []string, g *graph.Graph) []string {
	type nodeWithType struct {
		id   string
		node *graph.Node
	}

	nodes := make([]nodeWithType, 0, len(nodeIDs))
	for _, id := range nodeIDs {
		if node, exists := g.Nodes[id]; exists {
			nodes = append(nodes, nodeWithType{id: id, node: node})
		}
	}

	// Sort by resource type priority, then by name
	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].node.ResourceType != nodes[j].node.ResourceType {
			return getResourceTypePriority(nodes[i].node.ResourceType) <
				getResourceTypePriority(nodes[j].node.ResourceType)
		}
		return nodes[i].node.Name < nodes[j].node.Name
	})

	result := make([]string, len(nodes))
	for i, n := range nodes {
		result[i] = n.id
	}
	return result
}

// getResourceTypePriority returns priority for resource type ordering
func getResourceTypePriority(rt parser.ResourceType) int {
	priorities := map[parser.ResourceType]int{
		parser.ResourceTypeNetwork:      1,
		parser.ResourceTypeSecurity:     2,
		parser.ResourceTypeDNS:          3,
		parser.ResourceTypeCertificate:  4,
		parser.ResourceTypeLoadBalancer: 5,
		parser.ResourceTypeCompute:      6,
		parser.ResourceTypeContainer:    7,
		parser.ResourceTypeDatabase:     8,
		parser.ResourceTypeStorage:      9,
		parser.ResourceTypeCDN:          10,
		parser.ResourceTypeSecret:       11,
	}

	if p, exists := priorities[rt]; exists {
		return p
	}
	return 99
}

// minimizeCrossings uses barycenter heuristic to reduce edge crossings
func (il *ImprovedLayout) minimizeCrossings(layers [][]string, g *graph.Graph) {
	// Multiple passes for better results
	for pass := 0; pass < 3; pass++ {
		// Forward pass (top to bottom)
		for i := 1; i < len(layers); i++ {
			il.reorderLayerByBarycenter(layers, i, g, true)
		}

		// Backward pass (bottom to top)
		for i := len(layers) - 2; i >= 0; i-- {
			il.reorderLayerByBarycenter(layers, i, g, false)
		}
	}
}

// reorderLayerByBarycenter reorders a layer to minimize crossings
func (il *ImprovedLayout) reorderLayerByBarycenter(layers [][]string, layerIdx int, g *graph.Graph, forward bool) {
	if layerIdx < 0 || layerIdx >= len(layers) {
		return // Safety check
	}

	// Check if we have an adjacent layer to work with
	if forward && layerIdx == 0 {
		return // No previous layer to compare with
	}
	if !forward && layerIdx == len(layers)-1 {
		return // No next layer to compare with
	}

	type nodeWithPos struct {
		id       string
		position float64
	}

	layer := layers[layerIdx]
	positions := make([]nodeWithPos, len(layer))

	for i, nodeID := range layer {
		// Calculate barycenter (average position of connected nodes in adjacent layer)
		var sum float64
		var count int

		for _, edge := range g.Edges {
			var connectedID string
			var isConnected bool

			if forward && edge.To.ID == nodeID {
				connectedID = edge.From.ID
				isConnected = true
			} else if !forward && edge.From.ID == nodeID {
				connectedID = edge.To.ID
				isConnected = true
			}

			if isConnected {
				// Find position of connected node in adjacent layer
				var adjacentLayer []string
				if forward {
					adjacentLayer = layers[layerIdx-1]
				} else {
					adjacentLayer = layers[layerIdx+1]
				}

				for pos, id := range adjacentLayer {
					if id == connectedID {
						sum += float64(pos)
						count++
						break
					}
				}
			}
		}

		if count > 0 {
			positions[i] = nodeWithPos{id: nodeID, position: sum / float64(count)}
		} else {
			positions[i] = nodeWithPos{id: nodeID, position: float64(i)}
		}
	}

	// Sort by barycenter position
	sort.Slice(positions, func(i, j int) bool {
		return positions[i].position < positions[j].position
	})

	// Update layer
	for i, np := range positions {
		layers[layerIdx][i] = np.id
	}
}

// assignCoordinatesWithSpacing assigns coordinates with proper spacing
func (il *ImprovedLayout) assignCoordinatesWithSpacing(layers [][]string, direction string,
	nodeWidth, nodeHeight, hSpacing, vSpacing float64) {

	maxNodesInLayer := 0
	for _, layer := range layers {
		if len(layer) > maxNodesInLayer {
			maxNodesInLayer = len(layer)
		}
	}

	for layerIdx, layer := range layers {
		layerWidth := float64(len(layer)-1)*hSpacing + float64(len(layer))*nodeWidth
		startOffset := (float64(maxNodesInLayer)*nodeWidth + float64(maxNodesInLayer-1)*hSpacing - layerWidth) / 2

		for nodeIdx, nodeID := range layer {
			node := &NodeLayout{
				Width:  nodeWidth,
				Height: nodeHeight,
				Layer:  layerIdx,
			}

			var x, y float64

			switch direction {
			case "TB":
				x = startOffset + float64(nodeIdx)*(nodeWidth+hSpacing)
				y = float64(layerIdx) * (nodeHeight + vSpacing)
			case "BT":
				x = startOffset + float64(nodeIdx)*(nodeWidth+hSpacing)
				y = float64(len(layers)-1-layerIdx) * (nodeHeight + vSpacing)
			case "LR":
				x = float64(layerIdx) * (nodeWidth + hSpacing)
				y = startOffset + float64(nodeIdx)*(nodeHeight+vSpacing)
			case "RL":
				x = float64(len(layers)-1-layerIdx) * (nodeWidth + hSpacing)
				y = startOffset + float64(nodeIdx)*(nodeHeight+vSpacing)
			default:
				x = startOffset + float64(nodeIdx)*(nodeWidth+hSpacing)
				y = float64(layerIdx) * (nodeHeight + vSpacing)
			}

			node.Position = Point{X: x, Y: y}
			il.Nodes[nodeID] = node
			il.nodesByLayer[layerIdx] = append(il.nodesByLayer[layerIdx], node)
		}
	}

	// Calculate dimensions
	maxX, maxY := 0.0, 0.0
	for _, node := range il.Nodes {
		if node.Position.X+node.Width > maxX {
			maxX = node.Position.X + node.Width
		}
		if node.Position.Y+node.Height > maxY {
			maxY = node.Position.Y + node.Height
		}
	}

	il.Width = maxX + hSpacing
	il.Height = maxY + vSpacing
}

// resolveOverlaps detects and resolves any remaining overlaps
func (il *ImprovedLayout) resolveOverlaps(nodeWidth, nodeHeight float64) {
	// Simple overlap detection and resolution
	nodes := make([]*NodeLayout, 0, len(il.Nodes))
	for _, node := range il.Nodes {
		nodes = append(nodes, node)
	}

	// Check for overlaps and adjust
	for i := 0; i < len(nodes); i++ {
		for j := i + 1; j < len(nodes); j++ {
			if il.nodesOverlap(nodes[i], nodes[j]) {
				// Push nodes apart
				il.separateNodes(nodes[i], nodes[j], nodeWidth*0.2)
			}
		}
	}
}

// nodesOverlap checks if two nodes overlap
func (il *ImprovedLayout) nodesOverlap(n1, n2 *NodeLayout) bool {
	margin := 10.0 // Minimum space between nodes

	return !(n1.Position.X+n1.Width+margin < n2.Position.X ||
		n2.Position.X+n2.Width+margin < n1.Position.X ||
		n1.Position.Y+n1.Height+margin < n2.Position.Y ||
		n2.Position.Y+n2.Height+margin < n1.Position.Y)
}

// separateNodes moves nodes apart if they overlap
func (il *ImprovedLayout) separateNodes(n1, n2 *NodeLayout, distance float64) {
	// Calculate direction to move
	dx := n2.Position.X - n1.Position.X
	dy := n2.Position.Y - n1.Position.Y
	dist := math.Sqrt(dx*dx + dy*dy)

	if dist < 1.0 {
		dist = 1.0
	}

	// Normalize and move
	dx /= dist
	dy /= dist

	n2.Position.X += dx * distance
	n2.Position.Y += dy * distance
}

// calculateCurvedEdgePaths creates curved paths for edges
func (il *ImprovedLayout) calculateCurvedEdgePaths(g *graph.Graph) {
	for _, edge := range g.Edges {
		fromNode := il.Nodes[edge.From.ID]
		toNode := il.Nodes[edge.To.ID]

		if fromNode == nil || toNode == nil {
			continue
		}

		edgeLayout := &EdgeLayout{
			Edge:   edge,
			Points: il.calculateBezierCurve(fromNode, toNode),
		}

		il.Edges = append(il.Edges, edgeLayout)
	}
}

// calculateBezierCurve creates a smooth Bezier curve between nodes
func (il *ImprovedLayout) calculateBezierCurve(from, to *NodeLayout) []Point {
	// Connection points
	var startPoint, endPoint Point

	switch il.Direction {
	case "TB":
		startPoint = Point{X: from.Position.X + from.Width/2, Y: from.Position.Y + from.Height}
		endPoint = Point{X: to.Position.X + to.Width/2, Y: to.Position.Y}
	case "BT":
		startPoint = Point{X: from.Position.X + from.Width/2, Y: from.Position.Y}
		endPoint = Point{X: to.Position.X + to.Width/2, Y: to.Position.Y + to.Height}
	case "LR":
		startPoint = Point{X: from.Position.X + from.Width, Y: from.Position.Y + from.Height/2}
		endPoint = Point{X: to.Position.X, Y: to.Position.Y + to.Height/2}
	case "RL":
		startPoint = Point{X: from.Position.X, Y: from.Position.Y + from.Height/2}
		endPoint = Point{X: to.Position.X + to.Width, Y: to.Position.Y + to.Height/2}
	default:
		startPoint = Point{X: from.Position.X + from.Width/2, Y: from.Position.Y + from.Height}
		endPoint = Point{X: to.Position.X + to.Width/2, Y: to.Position.Y}
	}

	// Check if nodes are far apart - use curved line
	dx := endPoint.X - startPoint.X
	dy := endPoint.Y - startPoint.Y
	distance := math.Sqrt(dx*dx + dy*dy)

	// If very close or aligned, use straight line
	if distance < 100 || (math.Abs(dx) < 10 && il.Direction == "TB") ||
		(math.Abs(dy) < 10 && il.Direction == "LR") {
		return []Point{startPoint, endPoint}
	}

	// Create Bezier curve control points
	var cp1, cp2 Point

	switch il.Direction {
	case "TB", "BT":
		// Vertical layout - curve sideways
		curveStrength := math.Min(math.Abs(dy)*0.4, 80.0)
		cp1 = Point{X: startPoint.X, Y: startPoint.Y + curveStrength}
		cp2 = Point{X: endPoint.X, Y: endPoint.Y - curveStrength}
	case "LR", "RL":
		// Horizontal layout - curve vertically
		curveStrength := math.Min(math.Abs(dx)*0.4, 80.0)
		cp1 = Point{X: startPoint.X + curveStrength, Y: startPoint.Y}
		cp2 = Point{X: endPoint.X - curveStrength, Y: endPoint.Y}
	default:
		curveStrength := math.Min(math.Abs(dy)*0.4, 80.0)
		cp1 = Point{X: startPoint.X, Y: startPoint.Y + curveStrength}
		cp2 = Point{X: endPoint.X, Y: endPoint.Y - curveStrength}
	}

	// Generate smooth Bezier curve points
	points := []Point{startPoint}
	steps := 20

	for i := 1; i < steps; i++ {
		t := float64(i) / float64(steps)
		point := il.cubicBezier(startPoint, cp1, cp2, endPoint, t)
		points = append(points, point)
	}

	points = append(points, endPoint)
	return points
}

// cubicBezier calculates a point on a cubic Bezier curve
func (il *ImprovedLayout) cubicBezier(p0, p1, p2, p3 Point, t float64) Point {
	t2 := t * t
	t3 := t2 * t
	mt := 1 - t
	mt2 := mt * mt
	mt3 := mt2 * mt

	return Point{
		X: mt3*p0.X + 3*mt2*t*p1.X + 3*mt*t2*p2.X + t3*p3.X,
		Y: mt3*p0.Y + 3*mt2*t*p1.Y + 3*mt*t2*p2.Y + t3*p3.Y,
	}
}
