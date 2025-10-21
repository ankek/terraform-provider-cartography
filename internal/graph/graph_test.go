package graph

import (
	"context"
	"testing"

	"github.com/ankek/terraform-provider-cartography/internal/parser"
)

func TestBuildGraph(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		resources []parser.Resource
		wantNodes int
		wantEdges int
	}{
		{
			name:      "empty resources",
			resources: []parser.Resource{},
			wantNodes: 0,
			wantEdges: 0,
		},
		{
			name: "single resource",
			resources: []parser.Resource{
				{
					ID:       "aws_instance.web",
					Type:     "aws_instance",
					Name:     "web",
					Provider: "aws",
					Attributes: map[string]interface{}{
						"instance_type": "t2.micro",
					},
				},
			},
			wantNodes: 1,
			wantEdges: 0,
		},
		{
			name: "resources with dependency",
			resources: []parser.Resource{
				{
					ID:       "aws_instance.web",
					Type:     "aws_instance",
					Name:     "web",
					Provider: "aws",
					Dependencies: []string{"aws_security_group.web"},
				},
				{
					ID:       "aws_security_group.web",
					Type:     "aws_security_group",
					Name:     "web",
					Provider: "aws",
				},
			},
			wantNodes: 2,
			wantEdges: 1,
		},
		{
			name: "filter out non-infrastructure resources",
			resources: []parser.Resource{
				{
					ID:       "aws_instance.web",
					Type:     "aws_instance",
					Name:     "web",
					Provider: "aws",
				},
				{
					ID:       "local_file.config",
					Type:     "local_file",
					Name:     "config",
					Provider: "local",
				},
				{
					ID:       "tls_private_key.example",
					Type:     "tls_private_key",
					Name:     "example",
					Provider: "tls",
				},
			},
			wantNodes: 1, // Only aws_instance should be included
			wantEdges: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := BuildGraph(ctx, tt.resources)

			if len(g.Nodes) != tt.wantNodes {
				t.Errorf("BuildGraph() got %d nodes, want %d", len(g.Nodes), tt.wantNodes)
			}

			if len(g.Edges) != tt.wantEdges {
				t.Errorf("BuildGraph() got %d edges, want %d", len(g.Edges), tt.wantEdges)
			}
		})
	}
}

func TestBuildGraph_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	resources := []parser.Resource{
		{
			ID:       "aws_instance.web",
			Type:     "aws_instance",
			Name:     "web",
			Provider: "aws",
		},
	}

	g := BuildGraph(ctx, resources)

	// Graph should still be created but may be incomplete
	if g == nil {
		t.Error("BuildGraph() should return a graph even when context is cancelled")
	}
}

func TestFindNodeByAttributeValue(t *testing.T) {
	g := &Graph{
		Nodes:          make(map[string]*Node),
		attributeIndex: make(map[string]map[string]*Node),
	}

	// Create test nodes
	node1 := &Node{
		ID:   "aws_instance.web",
		Type: "aws_instance",
		Name: "web",
		Attributes: map[string]interface{}{
			"id":            "i-12345",
			"instance_type": "t2.micro",
		},
	}

	node2 := &Node{
		ID:   "aws_security_group.web",
		Type: "aws_security_group",
		Name: "web",
		Attributes: map[string]interface{}{
			"id": "sg-67890",
		},
	}

	g.Nodes["aws_instance.web"] = node1
	g.Nodes["aws_security_group.web"] = node2

	// Build index
	g.buildAttributeIndex()

	tests := []struct {
		name      string
		attrKey   string
		attrValue string
		wantNode  *Node
	}{
		{
			name:      "find by id - node1",
			attrKey:   "id",
			attrValue: "i-12345",
			wantNode:  node1,
		},
		{
			name:      "find by id - node2",
			attrKey:   "id",
			attrValue: "sg-67890",
			wantNode:  node2,
		},
		{
			name:      "find by instance_type",
			attrKey:   "instance_type",
			attrValue: "t2.micro",
			wantNode:  node1,
		},
		{
			name:      "not found",
			attrKey:   "id",
			attrValue: "nonexistent",
			wantNode:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := g.findNodeByAttributeValue(tt.attrKey, tt.attrValue)
			if got != tt.wantNode {
				t.Errorf("findNodeByAttributeValue() = %v, want %v", got, tt.wantNode)
			}
		})
	}
}

func TestInferRelationship(t *testing.T) {
	tests := []struct {
		name     string
		fromType parser.ResourceType
		toType   parser.ResourceType
		want     string
	}{
		{
			name:     "security to compute",
			fromType: parser.ResourceTypeSecurity,
			toType:   parser.ResourceTypeCompute,
			want:     "protects",
		},
		{
			name:     "security to load balancer",
			fromType: parser.ResourceTypeSecurity,
			toType:   parser.ResourceTypeLoadBalancer,
			want:     "filters",
		},
		{
			name:     "load balancer to compute",
			fromType: parser.ResourceTypeLoadBalancer,
			toType:   parser.ResourceTypeCompute,
			want:     "routes_to",
		},
		{
			name:     "network contains",
			fromType: parser.ResourceTypeNetwork,
			toType:   parser.ResourceTypeCompute,
			want:     "contains",
		},
		{
			name:     "compute to storage",
			fromType: parser.ResourceTypeCompute,
			toType:   parser.ResourceTypeStorage,
			want:     "uses_storage",
		},
		{
			name:     "compute to database",
			fromType: parser.ResourceTypeCompute,
			toType:   parser.ResourceTypeDatabase,
			want:     "connects_to_db",
		},
		{
			name:     "default relationship",
			fromType: parser.ResourceTypeCompute,
			toType:   parser.ResourceTypeCompute,
			want:     "depends_on",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			from := &Node{ResourceType: tt.fromType}
			to := &Node{ResourceType: tt.toType}

			got := inferRelationship(from, to)
			if got != tt.want {
				t.Errorf("inferRelationship() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractConnectionMetadata(t *testing.T) {
	tests := []struct {
		name       string
		from       *Node
		to         *Node
		wantEmpty  bool
		checkKey   string
		checkValue string
	}{
		{
			name: "no metadata",
			from: &Node{
				Provider:   "aws",
				Type:       "aws_instance",
				Attributes: map[string]interface{}{},
			},
			to:        &Node{},
			wantEmpty: true,
		},
		{
			name: "azure security rule with port",
			from: &Node{
				Provider: "azure",
				Type:     "azurerm_network_security_rule",
				Attributes: map[string]interface{}{
					"destination_port_range": "443",
					"protocol":               "Tcp",
				},
			},
			to:         &Node{},
			wantEmpty:  false,
			checkKey:   "port",
			checkValue: "443",
		},
		{
			name: "aws security group rule",
			from: &Node{
				Provider: "aws",
				Type:     "aws_security_group_rule",
				Attributes: map[string]interface{}{
					"from_port": "80",
					"protocol":  "tcp",
				},
			},
			to:         &Node{},
			wantEmpty:  false,
			checkKey:   "port",
			checkValue: "80",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractConnectionMetadata(tt.from, tt.to)

			if tt.wantEmpty {
				if len(got) != 0 {
					t.Errorf("extractConnectionMetadata() expected empty map, got %v", got)
				}
			} else {
				if val, ok := got[tt.checkKey]; !ok || val != tt.checkValue {
					t.Errorf("extractConnectionMetadata()[%s] = %v, want %v", tt.checkKey, val, tt.checkValue)
				}
			}
		})
	}
}

func TestEdgeDuplication(t *testing.T) {
	g := &Graph{
		Nodes: make(map[string]*Node),
		Edges: make([]*Edge, 0),
	}

	node1 := &Node{ID: "node1", Edges: make([]*Edge, 0)}
	node2 := &Node{ID: "node2", Edges: make([]*Edge, 0)}

	g.Nodes["node1"] = node1
	g.Nodes["node2"] = node2

	// Add edge twice
	g.addEdge(node1, node2, "depends_on", emptyMetadata)
	g.addEdge(node1, node2, "depends_on", emptyMetadata)

	// Should only have one edge
	if len(g.Edges) != 1 {
		t.Errorf("addEdge() created duplicate edge, got %d edges, want 1", len(g.Edges))
	}
}
