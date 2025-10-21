// Package graph provides functionality for building and analyzing resource dependency graphs.
// It creates directed graphs representing relationships between Terraform resources,
// with optimizations for efficient traversal and querying.
package graph

import (
	"context"
	"strings"

	"github.com/ankek/terraform-provider-cartography/internal/parser"
)

// Node represents a node in the resource graph
type Node struct {
	ID           string
	Type         string
	Name         string
	Provider     string
	ResourceType parser.ResourceType
	Attributes   map[string]interface{}
	Edges        []*Edge
}

// Edge represents a connection between two resources
type Edge struct {
	From         *Node
	To           *Node
	Relationship string            // e.g., "attached_to", "routes_to", "member_of"
	Metadata     map[string]string // Additional connection info (e.g., port numbers)
}

// Graph represents the complete resource graph of Terraform resources and their dependencies.
// Nodes represent resources (VMs, networks, databases, etc.) and edges represent
// relationships between them (depends_on, protects, routes_to, etc.).
type Graph struct {
	Nodes map[string]*Node
	Edges []*Edge
	// attributeIndex provides O(1) lookup of nodes by attribute values
	attributeIndex map[string]map[string]*Node
}

// edgeExists checks if an edge already exists between two nodes
func (g *Graph) edgeExists(from, to *Node) bool {
	for _, edge := range g.Edges {
		if edge.From.ID == from.ID && edge.To.ID == to.ID {
			return true
		}
	}
	return false
}

// addEdge adds an edge only if it doesn't already exist
func (g *Graph) addEdge(from, to *Node, relationship string, metadata map[string]string) {
	if g.edgeExists(from, to) {
		return // Don't add duplicate
	}

	edge := &Edge{
		From:         from,
		To:           to,
		Relationship: relationship,
		Metadata:     metadata,
	}

	g.Edges = append(g.Edges, edge)
	from.Edges = append(from.Edges, edge)
}

// BuildGraph creates a resource dependency graph from parsed Terraform resources.
// It filters out utility resources (TLS keys, local files, etc.) and builds
// a directed graph showing infrastructure dependencies.
//
// The function performs these steps:
//  1. Creates nodes for each cloud infrastructure resource
//  2. Adds edges based on explicit Terraform dependencies
//  3. Builds an attribute index for fast O(1) lookups
//  4. Detects implicit connections (e.g., security group to VM attachments)
//
// Returns a Graph ready for visualization. Respects context for cancellation.
func BuildGraph(ctx context.Context, resources []parser.Resource) *Graph {
	g := &Graph{
		Nodes:          make(map[string]*Node),
		Edges:          make([]*Edge, 0),
		attributeIndex: make(map[string]map[string]*Node),
	}

	// Create nodes (filter out non-infrastructure resources)
	for _, res := range resources {
		// Check context
		select {
		case <-ctx.Done():
			return g
		default:
		}
		// Skip non-cloud infrastructure resources (TLS keys, local files, etc.)
		if !parser.ShouldIncludeInDiagram(res) {
			continue
		}

		node := &Node{
			ID:           res.ID,
			Type:         res.Type,
			Name:         res.Name,
			Provider:     res.Provider,
			ResourceType: parser.GetResourceType(res.Type),
			Attributes:   res.Attributes,
			Edges:        make([]*Edge, 0),
		}
		g.Nodes[res.ID] = node
	}

	// Build attribute index for O(1) lookups (optimization for detectImplicitConnections)
	g.buildAttributeIndex()

	// Create edges based on dependencies
	for _, res := range resources {
		// Check context
		select {
		case <-ctx.Done():
			return g
		default:
		}

		fromNode := g.Nodes[res.ID]
		if fromNode == nil {
			continue
		}

		for _, depID := range res.Dependencies {
			toNode := g.Nodes[depID]
			if toNode == nil {
				continue
			}

			g.addEdge(fromNode, toNode, inferRelationship(fromNode, toNode), extractConnectionMetadata(fromNode, toNode))
		}
	}

	// Detect implicit connections (e.g., NSG rules referencing load balancers)
	g.detectImplicitConnections()

	return g
}

// buildAttributeIndex creates an index for fast O(1) node lookups by attribute values.
// This optimization reduces graph traversal from O(n²) to O(n) during implicit connection detection.
func (g *Graph) buildAttributeIndex() {
	for _, node := range g.Nodes {
		for attrKey, attrValue := range node.Attributes {
			if strValue, ok := attrValue.(string); ok {
				if g.attributeIndex[attrKey] == nil {
					g.attributeIndex[attrKey] = make(map[string]*Node)
				}
				g.attributeIndex[attrKey][strValue] = node
			}
		}
	}
}

// inferRelationship determines the type of relationship between two resources
func inferRelationship(from, to *Node) string {
	// Network security to compute/load balancer
	if from.ResourceType == parser.ResourceTypeSecurity {
		if to.ResourceType == parser.ResourceTypeCompute {
			return "protects"
		}
		if to.ResourceType == parser.ResourceTypeLoadBalancer {
			return "filters"
		}
	}

	// Load balancer to compute
	if from.ResourceType == parser.ResourceTypeLoadBalancer && to.ResourceType == parser.ResourceTypeCompute {
		return "routes_to"
	}

	// Network to subnet/security
	if from.ResourceType == parser.ResourceTypeNetwork {
		return "contains"
	}

	// Compute to storage/database
	if from.ResourceType == parser.ResourceTypeCompute {
		if to.ResourceType == parser.ResourceTypeStorage {
			return "uses_storage"
		}
		if to.ResourceType == parser.ResourceTypeDatabase {
			return "connects_to_db"
		}
	}

	return "depends_on"
}

// emptyMetadata is a shared empty map to avoid allocations.
// It's returned by extractConnectionMetadata when no metadata is found,
// reducing memory allocations in the hot path.
var emptyMetadata = map[string]string{}

// extractConnectionMetadata extracts metadata about the connection using safe attribute helpers.
// Returns a shared empty map if no metadata is found to avoid unnecessary allocations.
func extractConnectionMetadata(from, to *Node) map[string]string {
	var metadata map[string]string // nil initially

	// ensureMetadata lazily creates the metadata map only when needed
	ensureMetadata := func() {
		if metadata == nil {
			metadata = make(map[string]string)
		}
	}

	// Extract port information from security rules
	if from.Provider == "azure" && strings.Contains(from.Type, "security") {
		if port, ok := parser.GetStringAttribute(from.Attributes, "destination_port_range"); ok {
			ensureMetadata()
			metadata["port"] = port
		}
		if protocol, ok := parser.GetStringAttribute(from.Attributes, "protocol"); ok {
			ensureMetadata()
			metadata["protocol"] = protocol
		}
	}

	if from.Provider == "aws" && from.Type == "aws_security_group_rule" {
		if port, ok := parser.GetStringAttribute(from.Attributes, "from_port"); ok {
			ensureMetadata()
			metadata["port"] = port
		}
		if protocol, ok := parser.GetStringAttribute(from.Attributes, "protocol"); ok {
			ensureMetadata()
			metadata["protocol"] = protocol
		}
	}

	// Extract load balancer port information
	if strings.Contains(from.Type, "lb_rule") || strings.Contains(from.Type, "lb_listener") {
		if port, ok := parser.GetStringAttribute(from.Attributes, "frontend_port"); ok {
			ensureMetadata()
			metadata["frontend_port"] = port
		}
		if port, ok := parser.GetStringAttribute(from.Attributes, "backend_port"); ok {
			ensureMetadata()
			metadata["backend_port"] = port
		}
		if port, ok := parser.GetStringAttribute(from.Attributes, "port"); ok {
			ensureMetadata()
			metadata["port"] = port
		}
	}

	// DigitalOcean: Extract firewall rule ports - safely handle nested structures
	if from.Provider == "digitalocean" && from.Type == "digitalocean_firewall" {
		// Safely extract inbound rules
		if inboundRules, ok := from.Attributes["inbound_rule"].([]interface{}); ok && len(inboundRules) > 0 {
			if rule, ok := inboundRules[0].(map[string]interface{}); ok {
				if ports, ok := parser.GetStringAttribute(rule, "port_range"); ok {
					ensureMetadata()
					metadata["port"] = ports
				}
				if protocol, ok := parser.GetStringAttribute(rule, "protocol"); ok {
					ensureMetadata()
					metadata["protocol"] = protocol
				}
			}
		}
	}

	// DigitalOcean: Extract load balancer forwarding rules - safely
	if from.Provider == "digitalocean" && from.Type == "digitalocean_loadbalancer" {
		if forwardingRules, ok := from.Attributes["forwarding_rule"].([]interface{}); ok && len(forwardingRules) > 0 {
			if rule, ok := forwardingRules[0].(map[string]interface{}); ok {
				if port, ok := parser.GetStringAttribute(rule, "entry_port"); ok {
					ensureMetadata()
					metadata["frontend_port"] = port
				}
				if port, ok := parser.GetStringAttribute(rule, "target_port"); ok {
					ensureMetadata()
					metadata["backend_port"] = port
				}
				if protocol, ok := parser.GetStringAttribute(rule, "entry_protocol"); ok {
					ensureMetadata()
					metadata["protocol"] = protocol
				}
			}
		}
	}

	if metadata == nil {
		return emptyMetadata
	}
	return metadata
}

// detectImplicitConnections finds connections not explicitly in dependencies.
// Uses the attribute index for O(1) lookups instead of O(n) scans.
func (g *Graph) detectImplicitConnections() {
	// Azure: NSG to subnet associations
	for _, node := range g.Nodes {
		if node.Provider == "azure" && node.Type == "azurerm_subnet_network_security_group_association" {
			// Find subnet and NSG
			subnetID := getAttributeString(node.Attributes, "subnet_id")
			nsgID := getAttributeString(node.Attributes, "network_security_group_id")

			subnetNode := g.findNodeByAttributeValue("id", subnetID)
			nsgNode := g.findNodeByAttributeValue("id", nsgID)

			if subnetNode != nil && nsgNode != nil {
				g.addEdge(nsgNode, subnetNode, "protects", emptyMetadata)
			}
		}

		// AWS: Security group to instance
		if node.Provider == "aws" && node.Type == "aws_instance" {
			if sgIDs, ok := node.Attributes["vpc_security_group_ids"].([]interface{}); ok {
				for _, sgID := range sgIDs {
					if sgIDStr, ok := sgID.(string); ok {
						sgNode := g.findNodeByAttributeValue("id", sgIDStr)
						if sgNode != nil {
							g.addEdge(sgNode, node, "protects", emptyMetadata)
						}
					}
				}
			}
		}

		// DigitalOcean: Firewall to Droplet
		if node.Provider == "digitalocean" && node.Type == "digitalocean_droplet" {
			// Droplets can reference firewalls via tags or explicit firewall associations
			if dropletID := getAttributeString(node.Attributes, "id"); dropletID != "" {
				// Find firewalls that protect this droplet
				for _, fwNode := range g.Nodes {
					if fwNode.Provider == "digitalocean" && fwNode.Type == "digitalocean_firewall" {
						if dropletIDs, ok := fwNode.Attributes["droplet_ids"].([]interface{}); ok {
							for _, id := range dropletIDs {
								if idStr, ok := id.(string); ok && idStr == dropletID {
									g.addEdge(fwNode, node, "protects", emptyMetadata)
								}
							}
						}
					}
				}
			}
		}

		// DigitalOcean: Load Balancer to Droplets
		if node.Provider == "digitalocean" && node.Type == "digitalocean_loadbalancer" {
			if dropletIDs, ok := node.Attributes["droplet_ids"].([]interface{}); ok {
				for _, id := range dropletIDs {
					if idStr, ok := id.(string); ok {
						dropletNode := g.findNodeByAttributeValue("id", idStr)
						if dropletNode != nil {
							g.addEdge(node, dropletNode, "routes_to", emptyMetadata)
						}
					}
				}
			}
		}
	}
}

// Helper functions
func getAttributeString(attrs map[string]interface{}, key string) string {
	if val, ok := attrs[key]; ok {
		if strVal, ok := val.(string); ok {
			return strVal
		}
	}
	return ""
}

// findNodeByAttributeValue looks up a node by attribute value using the O(1) index.
// Falls back to O(n) scan if attribute is not indexed.
func (g *Graph) findNodeByAttributeValue(attrKey, attrValue string) *Node {
	// Try index lookup first (O(1))
	if index, ok := g.attributeIndex[attrKey]; ok {
		if node, found := index[attrValue]; found {
			return node
		}
	}

	// Fallback to linear scan for non-indexed attributes
	for _, node := range g.Nodes {
		if val := getAttributeString(node.Attributes, attrKey); val == attrValue {
			return node
		}
	}
	return nil
}
