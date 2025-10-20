package graph

import (
	"fmt"
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

// Graph represents the complete resource graph
type Graph struct {
	Nodes map[string]*Node
	Edges []*Edge
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

// BuildGraph creates a graph from parsed resources
func BuildGraph(resources []parser.Resource) *Graph {
	g := &Graph{
		Nodes: make(map[string]*Node),
		Edges: make([]*Edge, 0),
	}

	// Create nodes (filter out non-infrastructure resources)
	for _, res := range resources {
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

	// Create edges based on dependencies
	for _, res := range resources {
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

// extractConnectionMetadata extracts metadata about the connection
func extractConnectionMetadata(from, to *Node) map[string]string {
	metadata := make(map[string]string)

	// Extract port information from security rules
	if from.Provider == "azure" && strings.Contains(from.Type, "security") {
		if port, ok := from.Attributes["destination_port_range"].(string); ok {
			metadata["port"] = port
		}
		if protocol, ok := from.Attributes["protocol"].(string); ok {
			metadata["protocol"] = protocol
		}
	}

	if from.Provider == "aws" && from.Type == "aws_security_group_rule" {
		if port, ok := from.Attributes["from_port"].(float64); ok {
			metadata["port"] = fmt.Sprintf("%.0f", port)
		}
		if protocol, ok := from.Attributes["protocol"].(string); ok {
			metadata["protocol"] = protocol
		}
	}

	// Extract load balancer port information
	if strings.Contains(from.Type, "lb_rule") || strings.Contains(from.Type, "lb_listener") {
		if port, ok := from.Attributes["frontend_port"].(float64); ok {
			metadata["frontend_port"] = fmt.Sprintf("%.0f", port)
		}
		if port, ok := from.Attributes["backend_port"].(float64); ok {
			metadata["backend_port"] = fmt.Sprintf("%.0f", port)
		}
		if port, ok := from.Attributes["port"].(float64); ok {
			metadata["port"] = fmt.Sprintf("%.0f", port)
		}
	}

	// DigitalOcean: Extract firewall rule ports
	if from.Provider == "digitalocean" && from.Type == "digitalocean_firewall" {
		// Check inbound rules
		if inboundRules, ok := from.Attributes["inbound_rule"].([]interface{}); ok && len(inboundRules) > 0 {
			if rule, ok := inboundRules[0].(map[string]interface{}); ok {
				if ports, ok := rule["port_range"].(string); ok {
					metadata["port"] = ports
				}
				if protocol, ok := rule["protocol"].(string); ok {
					metadata["protocol"] = protocol
				}
			}
		}
	}

	// DigitalOcean: Extract load balancer forwarding rules
	if from.Provider == "digitalocean" && from.Type == "digitalocean_loadbalancer" {
		if forwardingRules, ok := from.Attributes["forwarding_rule"].([]interface{}); ok && len(forwardingRules) > 0 {
			if rule, ok := forwardingRules[0].(map[string]interface{}); ok {
				if port, ok := rule["entry_port"].(float64); ok {
					metadata["frontend_port"] = fmt.Sprintf("%.0f", port)
				}
				if port, ok := rule["target_port"].(float64); ok {
					metadata["backend_port"] = fmt.Sprintf("%.0f", port)
				}
				if protocol, ok := rule["entry_protocol"].(string); ok {
					metadata["protocol"] = protocol
				}
			}
		}
	}

	return metadata
}

// detectImplicitConnections finds connections not explicitly in dependencies
func (g *Graph) detectImplicitConnections() {
	// Azure: NSG to subnet associations
	for _, node := range g.Nodes {
		if node.Provider == "azure" && node.Type == "azurerm_subnet_network_security_group_association" {
			// Find subnet and NSG
			subnetID := getAttributeString(node.Attributes, "subnet_id")
			nsgID := getAttributeString(node.Attributes, "network_security_group_id")

			subnetNode := findNodeByAttributeValue(g.Nodes, "id", subnetID)
			nsgNode := findNodeByAttributeValue(g.Nodes, "id", nsgID)

			if subnetNode != nil && nsgNode != nil {
				g.addEdge(nsgNode, subnetNode, "protects", make(map[string]string))
			}
		}

		// AWS: Security group to instance
		if node.Provider == "aws" && node.Type == "aws_instance" {
			if sgIDs, ok := node.Attributes["vpc_security_group_ids"].([]interface{}); ok {
				for _, sgID := range sgIDs {
					if sgIDStr, ok := sgID.(string); ok {
						sgNode := findNodeByAttributeValue(g.Nodes, "id", sgIDStr)
						if sgNode != nil {
							g.addEdge(sgNode, node, "protects", make(map[string]string))
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
									g.addEdge(fwNode, node, "protects", make(map[string]string))
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
						dropletNode := findNodeByAttributeValue(g.Nodes, "id", idStr)
						if dropletNode != nil {
							g.addEdge(node, dropletNode, "routes_to", make(map[string]string))
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

func findNodeByAttributeValue(nodes map[string]*Node, attrKey, attrValue string) *Node {
	for _, node := range nodes {
		if val := getAttributeString(node.Attributes, attrKey); val == attrValue {
			return node
		}
	}
	return nil
}
