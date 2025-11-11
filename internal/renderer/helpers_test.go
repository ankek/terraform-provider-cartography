package renderer

import (
	"testing"

	"github.com/ankek/terraform-provider-cartography/internal/graph"
	"github.com/ankek/terraform-provider-cartography/internal/parser"
)

func TestFormatEdgeLabel(t *testing.T) {
	tests := []struct {
		name     string
		edge     *graph.Edge
		expected string
	}{
		{
			name: "with port and protocol",
			edge: &graph.Edge{
				Relationship: "connects",
				Metadata: map[string]string{
					"port":     "443",
					"protocol": "tcp",
				},
			},
			expected: "connects :443 tcp",
		},
		{
			name: "with port only",
			edge: &graph.Edge{
				Relationship: "connects",
				Metadata: map[string]string{
					"port": "80",
				},
			},
			expected: "connects :80",
		},
		{
			name: "with protocol only",
			edge: &graph.Edge{
				Relationship: "connects",
				Metadata: map[string]string{
					"protocol": "https",
				},
			},
			expected: "connects https",
		},
		{
			name: "no metadata",
			edge: &graph.Edge{
				Relationship: "depends_on",
				Metadata:     map[string]string{},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatEdgeLabel(tt.edge)
			if got != tt.expected {
				t.Errorf("formatEdgeLabel() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetNodeColor(t *testing.T) {
	tests := []struct {
		name         string
		resourceType parser.ResourceType
		expected     string
	}{
		{
			name:         "network resource",
			resourceType: parser.ResourceTypeNetwork,
			expected:     "#1E88E5",
		},
		{
			name:         "security resource",
			resourceType: parser.ResourceTypeSecurity,
			expected:     "#E53935",
		},
		{
			name:         "compute resource",
			resourceType: parser.ResourceTypeCompute,
			expected:     "#43A047",
		},
		{
			name:         "load balancer resource",
			resourceType: parser.ResourceTypeLoadBalancer,
			expected:     "#FB8C00",
		},
		{
			name:         "storage resource",
			resourceType: parser.ResourceTypeStorage,
			expected:     "#8E24AA",
		},
		{
			name:         "database resource",
			resourceType: parser.ResourceTypeDatabase,
			expected:     "#00ACC1",
		},
		{
			name:         "dns resource",
			resourceType: parser.ResourceTypeDNS,
			expected:     "#FDD835",
		},
		{
			name:         "certificate resource",
			resourceType: parser.ResourceTypeCertificate,
			expected:     "#7CB342",
		},
		{
			name:         "secret resource",
			resourceType: parser.ResourceTypeSecret,
			expected:     "#5E35B1",
		},
		{
			name:         "container resource",
			resourceType: parser.ResourceTypeContainer,
			expected:     "#039BE5",
		},
		{
			name:         "cdn resource",
			resourceType: parser.ResourceTypeCDN,
			expected:     "#F4511E",
		},
		{
			name:         "unknown resource",
			resourceType: parser.ResourceTypeUnknown,
			expected:     "#757575",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := &graph.Node{
				ResourceType: tt.resourceType,
			}
			got := getNodeColor(node)
			if got != tt.expected {
				t.Errorf("getNodeColor() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetResourceTypeName(t *testing.T) {
	tests := []struct {
		name         string
		resourceType string
		expected     string
	}{
		{
			name:         "azurerm resource",
			resourceType: "azurerm_virtual_machine",
			expected:     "Virtual Machine",
		},
		{
			name:         "aws resource",
			resourceType: "aws_instance",
			expected:     "Instance",
		},
		{
			name:         "google resource",
			resourceType: "google_compute_instance",
			expected:     "Compute Instance",
		},
		{
			name:         "digitalocean resource",
			resourceType: "digitalocean_droplet",
			expected:     "Droplet",
		},
		{
			name:         "no provider prefix",
			resourceType: "custom_resource",
			expected:     "Custom Resource",
		},
		{
			name:         "single word",
			resourceType: "resource",
			expected:     "Resource",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getResourceTypeName(tt.resourceType)
			if got != tt.expected {
				t.Errorf("getResourceTypeName() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "short string",
			input:    "hello",
			maxLen:   10,
			expected: "hello",
		},
		{
			name:     "exact length",
			input:    "hello",
			maxLen:   5,
			expected: "hello",
		},
		{
			name:     "long string",
			input:    "hello world this is a test",
			maxLen:   10,
			expected: "hello w...",
		},
		{
			name:     "very short max",
			input:    "hello",
			maxLen:   3,
			expected: "...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncate(tt.input, tt.maxLen)
			if got != tt.expected {
				t.Errorf("truncate() = %v, want %v", got, tt.expected)
			}
		})
	}
}
