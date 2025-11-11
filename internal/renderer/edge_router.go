package renderer

import (
	"math"

	"github.com/ankek/terraform-provider-cartography/internal/graph"
)

// EdgeRouter handles intelligent edge routing to prevent overlaps
type EdgeRouter struct {
	layout    *Layout
	edges     []*EdgeRoute
	nodeWidth float64
	nodeHeight float64
}

// EdgeRoute represents a routed edge with multiple segments
type EdgeRoute struct {
	edge     *graph.Edge
	segments []EdgeSegment
	offset   float64 // Horizontal offset for parallel edges
}

// EdgeSegment represents a segment of a routed edge
type EdgeSegment struct {
	start Point
	end   Point
	style string // "straight", "curve", "orthogonal"
}

// NewEdgeRouter creates a new edge router
func NewEdgeRouter(layout *Layout, nodeWidth, nodeHeight float64) *EdgeRouter {
	return &EdgeRouter{
		layout:     layout,
		edges:      make([]*EdgeRoute, 0),
		nodeWidth:  nodeWidth,
		nodeHeight: nodeHeight,
	}
}

// RouteEdges routes all edges to avoid overlaps
func (er *EdgeRouter) RouteEdges(g *graph.Graph) []*EdgeLayout {
	// First pass: identify parallel edges and assign offsets
	er.identifyParallelEdges(g)

	// Group edges by target node for connection point distribution
	edgesByTarget := make(map[string][]*graph.Edge)
	for _, edge := range g.Edges {
		edgesByTarget[edge.To.ID] = append(edgesByTarget[edge.To.ID], edge)
	}

	// Second pass: route each edge avoiding overlaps
	layouts := make([]*EdgeLayout, 0, len(g.Edges))

	for _, edge := range g.Edges {
		fromNode := er.layout.Nodes[edge.From.ID]
		toNode := er.layout.Nodes[edge.To.ID]

		if fromNode == nil || toNode == nil {
			continue
		}

		// Find if this edge has a route with offset
		var offset float64
		for _, route := range er.edges {
			if route.edge == edge {
				offset = route.offset
				break
			}
		}

		// Calculate connection point offset if multiple edges target same node
		connectionOffset := 0.0
		targetEdges := edgesByTarget[edge.To.ID]
		if len(targetEdges) > 1 {
			// Find this edge's index among edges to same target
			edgeIndex := -1
			for i, e := range targetEdges {
				if e == edge {
					edgeIndex = i
					break
				}
			}
			if edgeIndex >= 0 {
				// Distribute connection points across the target node's top edge
				// Center the distribution around the middle
				spacing := 30.0 // pixels between connection points
				totalWidth := float64(len(targetEdges)-1) * spacing
				connectionOffset = (float64(edgeIndex) * spacing) - (totalWidth / 2.0)
			}
		}

		// Route the edge with both offsets
		points := er.routeEdgeWithConnection(fromNode, toNode, offset, connectionOffset)

		layouts = append(layouts, &EdgeLayout{
			Edge:   edge,
			Points: points,
		})
	}

	return layouts
}

// identifyParallelEdges finds edges that connect the same nodes and assigns offsets
func (er *EdgeRouter) identifyParallelEdges(g *graph.Graph) {
	// Group edges by node pairs (considering both directions as same connection)
	edgeGroups := make(map[string][]*graph.Edge)
	seen := make(map[string]bool)

	for _, edge := range g.Edges {
		// Create normalized key (always smaller ID first to treat A->B and B->A as same)
		var key string
		if edge.From.ID < edge.To.ID {
			key = edge.From.ID + "-" + edge.To.ID
		} else {
			key = edge.To.ID + "-" + edge.From.ID
		}

		// Skip if we've already seen this connection
		edgeKey := edge.From.ID + "-" + edge.To.ID
		if seen[edgeKey] {
			continue
		}
		seen[edgeKey] = true

		edgeGroups[key] = append(edgeGroups[key], edge)
	}

	// Use only first edge for each connection (no parallel edges)
	for _, edges := range edgeGroups {
		// Only use the first edge for each unique connection
		er.edges = append(er.edges, &EdgeRoute{
			edge:   edges[0],
			offset: 0,
		})
	}
}

// routeEdgeWithConnection routes a single edge with path offset and connection point offset
func (er *EdgeRouter) routeEdgeWithConnection(from, to *NodeLayout, pathOffset, connectionOffset float64) []Point {
	// Determine connection points based on direction with connection offset
	startPoint, endPoint := er.getConnectionPointsWithOffset(from, to, connectionOffset)

	// Calculate distance and angle
	dx := endPoint.X - startPoint.X
	dy := endPoint.Y - startPoint.Y
	distance := math.Sqrt(dx*dx + dy*dy)

	// For very close nodes or aligned nodes, use straight line with offset
	if distance < 50 {
		return er.routeStraightWithOffset(startPoint, endPoint, pathOffset)
	}

	// Check if nodes are in same layer (might overlap)
	if from.Layer == to.Layer {
		// Use orthogonal routing to avoid overlap
		return er.routeOrthogonal(startPoint, endPoint, pathOffset, from, to)
	}

	// Check if direct line would pass through other nodes
	if er.wouldIntersectNodes(startPoint, endPoint, from, to) {
		// Use curved routing to avoid nodes
		return er.routeCurvedAvoidance(startPoint, endPoint, pathOffset, from, to)
	}

	// Default: smooth curved line with offset
	return er.routeCurvedWithOffset(startPoint, endPoint, pathOffset)
}

// getConnectionPointsWithOffset determines connection points with horizontal offset for the target
func (er *EdgeRouter) getConnectionPointsWithOffset(from, to *NodeLayout, connectionOffset float64) (Point, Point) {
	var startPoint, endPoint Point

	// Calculate centers
	fromCenter := Point{
		X: from.Position.X + from.Width/2,
		Y: from.Position.Y + from.Height/2,
	}
	toCenter := Point{
		X: to.Position.X + to.Width/2,
		Y: to.Position.Y + to.Height/2,
	}

	// Calculate angle between nodes
	angle := math.Atan2(toCenter.Y-fromCenter.Y, toCenter.X-fromCenter.X)

	// Arrow clearance - space between edge end and node border
	arrowClearance := 10.0

	// Determine exit/entry points based on angle
	switch er.layout.Direction {
	case "TB", "BT":
		// Vertical layout - prefer top/bottom connections
		if to.Position.Y > from.Position.Y+from.Height {
			// To is below From - connect from bottom to top with clearance
			// Apply horizontal offset to target connection point
			startPoint = Point{X: fromCenter.X, Y: from.Position.Y + from.Height}
			endPoint = Point{X: toCenter.X + connectionOffset, Y: to.Position.Y - arrowClearance}
		} else if to.Position.Y+to.Height < from.Position.Y {
			// To is above From - connect from top to bottom with clearance
			startPoint = Point{X: fromCenter.X, Y: from.Position.Y}
			endPoint = Point{X: toCenter.X + connectionOffset, Y: to.Position.Y + to.Height + arrowClearance}
		} else {
			// Side-by-side - use side connections with clearance
			if toCenter.X > fromCenter.X {
				startPoint = Point{X: from.Position.X + from.Width, Y: fromCenter.Y}
				endPoint = Point{X: to.Position.X - arrowClearance, Y: toCenter.Y}
			} else {
				startPoint = Point{X: from.Position.X, Y: fromCenter.Y}
				endPoint = Point{X: to.Position.X + to.Width + arrowClearance, Y: toCenter.Y}
			}
		}

	case "LR", "RL":
		// Horizontal layout - prefer left/right connections
		if to.Position.X > from.Position.X+from.Width {
			// To is right of From - add clearance and vertical offset
			startPoint = Point{X: from.Position.X + from.Width, Y: fromCenter.Y}
			endPoint = Point{X: to.Position.X - arrowClearance, Y: toCenter.Y + connectionOffset}
		} else if to.Position.X+to.Width < from.Position.X {
			// To is left of From - add clearance and vertical offset
			startPoint = Point{X: from.Position.X, Y: fromCenter.Y}
			endPoint = Point{X: to.Position.X + to.Width + arrowClearance, Y: toCenter.Y + connectionOffset}
		} else {
			// Stacked - use top/bottom connections with clearance and horizontal offset
			if toCenter.Y > fromCenter.Y {
				startPoint = Point{X: fromCenter.X, Y: from.Position.Y + from.Height}
				endPoint = Point{X: toCenter.X + connectionOffset, Y: to.Position.Y - arrowClearance}
			} else {
				startPoint = Point{X: fromCenter.X, Y: from.Position.Y}
				endPoint = Point{X: toCenter.X + connectionOffset, Y: to.Position.Y + to.Height + arrowClearance}
			}
		}

	default:
		// Default to angle-based connection with clearance
		radiusFrom := math.Min(from.Width, from.Height)/2 + arrowClearance
		radiusTo := math.Min(to.Width, to.Height)/2 + arrowClearance
		startPoint = Point{
			X: fromCenter.X + radiusFrom*math.Cos(angle),
			Y: fromCenter.Y + radiusFrom*math.Sin(angle),
		}
		endPoint = Point{
			X: toCenter.X - radiusTo*math.Cos(angle),
			Y: toCenter.Y - radiusTo*math.Sin(angle),
		}
	}

	return startPoint, endPoint
}

// routeStraightWithOffset creates a straight line with horizontal offset
func (er *EdgeRouter) routeStraightWithOffset(start, end Point, offset float64) []Point {
	if offset == 0 {
		return []Point{start, end}
	}

	// Calculate perpendicular offset
	dx := end.X - start.X
	dy := end.Y - start.Y
	length := math.Sqrt(dx*dx + dy*dy)

	if length < 0.1 {
		return []Point{start, end}
	}

	// Perpendicular vector
	perpX := -dy / length * offset
	perpY := dx / length * offset

	// Create offset path
	midPoint := Point{
		X: (start.X + end.X) / 2,
		Y: (start.Y + end.Y) / 2,
	}

	offsetMid := Point{
		X: midPoint.X + perpX,
		Y: midPoint.Y + perpY,
	}

	return []Point{start, offsetMid, end}
}

// routeOrthogonal creates orthogonal (right-angle) routing
func (er *EdgeRouter) routeOrthogonal(start, end Point, offset float64, from, to *NodeLayout) []Point {
	points := []Point{start}

	// Add offset to avoid overlap
	offsetAmount := offset

	switch er.layout.Direction {
	case "TB", "BT":
		// Vertical layout - go down, across, then to target
		midY := (start.Y + end.Y) / 2
		points = append(points,
			Point{X: start.X, Y: midY},
			Point{X: end.X + offsetAmount, Y: midY},
			Point{X: end.X, Y: end.Y},
		)

	case "LR", "RL":
		// Horizontal layout - go right, down, then to target
		midX := (start.X + end.X) / 2
		points = append(points,
			Point{X: midX, Y: start.Y},
			Point{X: midX, Y: end.Y + offsetAmount},
			Point{X: end.X, Y: end.Y},
		)

	default:
		// Default orthogonal
		points = append(points,
			Point{X: end.X, Y: start.Y},
			end,
		)
	}

	return points
}

// routeCurvedWithOffset creates a curved path with offset for parallel edges
func (er *EdgeRouter) routeCurvedWithOffset(start, end Point, offset float64) []Point {
	if offset == 0 {
		// No offset - use standard Bezier curve
		return er.generateBezierCurve(start, end)
	}

	// Calculate control points with offset
	dx := end.X - start.X
	dy := end.Y - start.Y
	length := math.Sqrt(dx*dx + dy*dy)

	if length < 0.1 {
		return []Point{start, end}
	}

	// Perpendicular offset vector
	perpX := -dy / length * offset
	perpY := dx / length * offset

	// Control points with offset
	var cp1, cp2 Point

	switch er.layout.Direction {
	case "TB", "BT":
		curveStrength := math.Min(math.Abs(dy)*0.4, 100.0)
		cp1 = Point{
			X: start.X + perpX,
			Y: start.Y + curveStrength,
		}
		cp2 = Point{
			X: end.X + perpX,
			Y: end.Y - curveStrength,
		}

	case "LR", "RL":
		curveStrength := math.Min(math.Abs(dx)*0.4, 100.0)
		cp1 = Point{
			X: start.X + curveStrength,
			Y: start.Y + perpY,
		}
		cp2 = Point{
			X: end.X - curveStrength,
			Y: end.Y + perpY,
		}

	default:
		curveStrength := math.Min(length*0.3, 80.0)
		cp1 = Point{X: start.X, Y: start.Y + curveStrength}
		cp2 = Point{X: end.X, Y: end.Y - curveStrength}
	}

	return er.cubicBezierPoints(start, cp1, cp2, end, 25)
}

// routeCurvedAvoidance routes around obstacles
func (er *EdgeRouter) routeCurvedAvoidance(start, end Point, offset float64, from, to *NodeLayout) []Point {
	// Find intermediate waypoint to avoid nodes
	waypoint := er.findAvoidanceWaypoint(start, end, from, to)

	// Create two curves: start->waypoint and waypoint->end
	curve1 := er.routeCurvedWithOffset(start, waypoint, offset)
	curve2 := er.routeCurvedWithOffset(waypoint, end, offset)

	// Combine curves
	points := curve1
	points = append(points, curve2[1:]...)
	return points
}

// findAvoidanceWaypoint finds a point that avoids obstacles
func (er *EdgeRouter) findAvoidanceWaypoint(start, end Point, from, to *NodeLayout) Point {
	// Simple strategy: go around to the side
	midX := (start.X + end.X) / 2
	midY := (start.Y + end.Y) / 2

	// Offset to the side to avoid direct path
	sideOffset := 80.0

	switch er.layout.Direction {
	case "TB", "BT":
		// Go to the side
		if start.X < end.X {
			return Point{X: midX + sideOffset, Y: midY}
		}
		return Point{X: midX - sideOffset, Y: midY}

	case "LR", "RL":
		// Go up or down
		if start.Y < end.Y {
			return Point{X: midX, Y: midY + sideOffset}
		}
		return Point{X: midX, Y: midY - sideOffset}

	default:
		return Point{X: midX + sideOffset, Y: midY}
	}
}

// wouldIntersectNodes checks if a straight line would intersect other nodes
func (er *EdgeRouter) wouldIntersectNodes(start, end Point, from, to *NodeLayout) bool {
	for _, node := range er.layout.Nodes {
		if node == from || node == to {
			continue
		}

		// Check if line intersects node's bounding box (with margin)
		margin := 20.0
		if er.lineIntersectsRect(start, end,
			node.Position.X-margin, node.Position.Y-margin,
			node.Position.X+node.Width+margin, node.Position.Y+node.Height+margin) {
			return true
		}
	}
	return false
}

// lineIntersectsRect checks if a line segment intersects a rectangle
func (er *EdgeRouter) lineIntersectsRect(p1, p2 Point, x1, y1, x2, y2 float64) bool {
	// Simple AABB line intersection test
	minX, maxX := math.Min(p1.X, p2.X), math.Max(p1.X, p2.X)
	minY, maxY := math.Min(p1.Y, p2.Y), math.Max(p1.Y, p2.Y)

	// Check if line's bounding box intersects rect
	if maxX < x1 || minX > x2 || maxY < y1 || minY > y2 {
		return false
	}

	// More detailed intersection test
	// Check if line passes through rectangle
	dx := p2.X - p1.X
	dy := p2.Y - p1.Y

	if dx == 0 && dy == 0 {
		// Point, not a line
		return p1.X >= x1 && p1.X <= x2 && p1.Y >= y1 && p1.Y <= y2
	}

	// Check intersection with rect edges
	t1 := (x1 - p1.X) / dx
	t2 := (x2 - p1.X) / dx
	t3 := (y1 - p1.Y) / dy
	t4 := (y2 - p1.Y) / dy

	tmin := math.Max(math.Min(t1, t2), math.Min(t3, t4))
	tmax := math.Min(math.Max(t1, t2), math.Max(t3, t4))

	return tmin <= tmax && tmax >= 0 && tmin <= 1
}

// generateBezierCurve creates a standard Bezier curve
func (er *EdgeRouter) generateBezierCurve(start, end Point) []Point {
	dx := end.X - start.X
	dy := end.Y - start.Y
	distance := math.Sqrt(dx*dx + dy*dy)

	if distance < 100 {
		return []Point{start, end}
	}

	var cp1, cp2 Point

	switch er.layout.Direction {
	case "TB", "BT":
		curveStrength := math.Min(math.Abs(dy)*0.4, 100.0)
		cp1 = Point{X: start.X, Y: start.Y + curveStrength}
		cp2 = Point{X: end.X, Y: end.Y - curveStrength}

	case "LR", "RL":
		curveStrength := math.Min(math.Abs(dx)*0.4, 100.0)
		cp1 = Point{X: start.X + curveStrength, Y: start.Y}
		cp2 = Point{X: end.X - curveStrength, Y: end.Y}

	default:
		curveStrength := math.Min(math.Abs(dy)*0.4, 80.0)
		cp1 = Point{X: start.X, Y: start.Y + curveStrength}
		cp2 = Point{X: end.X, Y: end.Y - curveStrength}
	}

	return er.cubicBezierPoints(start, cp1, cp2, end, 25)
}

// cubicBezierPoints generates points along a cubic Bezier curve
func (er *EdgeRouter) cubicBezierPoints(p0, p1, p2, p3 Point, steps int) []Point {
	points := []Point{p0}

	for i := 1; i < steps; i++ {
		t := float64(i) / float64(steps)
		point := er.cubicBezier(p0, p1, p2, p3, t)
		points = append(points, point)
	}

	points = append(points, p3)
	return points
}

// cubicBezier calculates a point on a cubic Bezier curve
func (er *EdgeRouter) cubicBezier(p0, p1, p2, p3 Point, t float64) Point {
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
