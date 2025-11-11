package renderer

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ankek/terraform-provider-cartography/internal/graph"
	"github.com/ankek/terraform-provider-cartography/internal/parser"
)

func TestRenderDiagram(t *testing.T) {
	// Create a simple graph for testing
	g := &graph.Graph{
		Nodes: map[string]*graph.Node{
			"aws_instance.web": {
				ID:       "aws_instance.web",
				Type:     "aws_instance",
				Name:     "web",
				Provider: "aws",
				Attributes: map[string]interface{}{
					"id":            "i-12345",
					"instance_type": "t2.micro",
				},
			},
			"aws_vpc.main": {
				ID:       "aws_vpc.main",
				Type:     "aws_vpc",
				Name:     "main",
				Provider: "aws",
				Attributes: map[string]interface{}{
					"id":         "vpc-12345",
					"cidr_block": "10.0.0.0/16",
				},
			},
		},
		Edges: []*graph.Edge{
			{
				Relationship: "member_of",
			},
		},
	}

	// Link edge to nodes
	g.Edges[0].From = g.Nodes["aws_instance.web"]
	g.Edges[0].To = g.Nodes["aws_vpc.main"]

	tests := []struct {
		name    string
		opts    RenderOptions
		wantErr bool
	}{
		{
			name: "SVG format",
			opts: RenderOptions{
				Format:        "svg",
				Direction:     "TB",
				IncludeLabels: true,
				Title:         "Test Infrastructure",
				UseIcons:      false,
			},
			wantErr: false,
		},
		{
			name: "SVG with icons",
			opts: RenderOptions{
				Format:        "svg",
				Direction:     "LR",
				IncludeLabels: true,
				Title:         "Test Infrastructure",
				UseIcons:      true,
			},
			wantErr: false,
		},
		{
			name: "SVG without labels",
			opts: RenderOptions{
				Format:        "svg",
				Direction:     "TB",
				IncludeLabels: false,
				Title:         "Minimal Diagram",
				UseIcons:      false,
			},
			wantErr: false,
		},
		{
			name: "unsupported format",
			opts: RenderOptions{
				Format:    "pdf",
				Direction: "TB",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			outputPath := filepath.Join(tmpDir, "diagram.svg")

			ctx := context.Background()
			err := RenderDiagram(ctx, g, outputPath, tt.opts)

			if (err != nil) != tt.wantErr {
				t.Errorf("RenderDiagram() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify file was created
				if _, err := os.Stat(outputPath); os.IsNotExist(err) {
					t.Errorf("RenderDiagram() did not create output file: %s", outputPath)
				}

				// Verify file has content
				content, err := os.ReadFile(outputPath)
				if err != nil {
					t.Errorf("Failed to read output file: %v", err)
				}
				if len(content) == 0 {
					t.Error("RenderDiagram() created empty file")
				}

				// Verify SVG content
				if tt.opts.Format == "svg" {
					contentStr := string(content)
					if len(contentStr) < 100 {
						t.Error("SVG content seems too short")
					}
					// SVG should contain basic structure
					if tt.opts.IncludeLabels && tt.opts.Title != "" {
						// Title should appear somewhere in the SVG
						if len(tt.opts.Title) > 0 {
							// Just verify we have substantial content
							if len(contentStr) < 500 {
								t.Error("SVG with title and labels should have more content")
							}
						}
					}
				}
			}
		})
	}
}

func TestRenderDiagram_ContextCancellation(t *testing.T) {
	g := &graph.Graph{
		Nodes: map[string]*graph.Node{
			"aws_instance.web": {
				ID:       "aws_instance.web",
				Type:     "aws_instance",
				Name:     "web",
				Provider: "aws",
			},
		},
		Edges: []*graph.Edge{},
	}

	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "diagram.svg")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	opts := RenderOptions{
		Format:    "svg",
		Direction: "TB",
	}

	err := RenderDiagram(ctx, g, outputPath, opts)
	if err != context.Canceled {
		t.Errorf("RenderDiagram() with cancelled context got error = %v, want context.Canceled", err)
	}
}

func TestExportDiagram(t *testing.T) {
	g := &graph.Graph{
		Nodes: map[string]*graph.Node{
			"azurerm_resource_group.rg": {
				ID:       "azurerm_resource_group.rg",
				Type:     "azurerm_resource_group",
				Name:     "rg",
				Provider: "azure",
			},
		},
		Edges: []*graph.Edge{},
	}

	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "diagram.svg")

	ctx := context.Background()
	opts := RenderOptions{
		Format:        "svg",
		Direction:     "TB",
		IncludeLabels: true,
		Title:         "Azure Infrastructure",
		UseIcons:      false,
	}

	err := ExportDiagram(ctx, g, outputPath, opts)
	if err != nil {
		t.Errorf("ExportDiagram() error = %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("ExportDiagram() did not create output file")
	}
}

func TestRenderDiagram_EmptyGraph(t *testing.T) {
	g := &graph.Graph{
		Nodes: map[string]*graph.Node{},
		Edges: []*graph.Edge{},
	}

	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "diagram.svg")

	ctx := context.Background()
	opts := RenderOptions{
		Format:    "svg",
		Direction: "TB",
	}

	err := RenderDiagram(ctx, g, outputPath, opts)
	// Should handle empty graph gracefully
	if err != nil {
		t.Errorf("RenderDiagram() with empty graph error = %v", err)
	}
}

func TestRenderDiagram_MultipleDirections(t *testing.T) {
	g := &graph.Graph{
		Nodes: map[string]*graph.Node{
			"aws_instance.web": {
				ID:           "aws_instance.web",
				Type:         "aws_instance",
				Name:         "web",
				Provider:     "aws",
				ResourceType: parser.ResourceTypeCompute,
			},
			"aws_vpc.main": {
				ID:           "aws_vpc.main",
				Type:         "aws_vpc",
				Name:         "main",
				Provider:     "aws",
				ResourceType: parser.ResourceTypeNetwork,
			},
		},
		Edges: []*graph.Edge{},
	}

	directions := []string{"TB", "LR", "BT", "RL"}

	for _, direction := range directions {
		t.Run(direction, func(t *testing.T) {
			tmpDir := t.TempDir()
			outputPath := filepath.Join(tmpDir, "diagram.svg")

			ctx := context.Background()
			opts := RenderOptions{
				Format:        "svg",
				Direction:     direction,
				IncludeLabels: true,
				UseIcons:      false,
			}

			err := RenderDiagram(ctx, g, outputPath, opts)
			if err != nil {
				t.Errorf("RenderDiagram() with direction %s error = %v", direction, err)
			}

			// Verify file was created
			if _, err := os.Stat(outputPath); os.IsNotExist(err) {
				t.Errorf("RenderDiagram() with direction %s did not create output file", direction)
			}
		})
	}
}

func TestRenderDiagram_LargeGraph(t *testing.T) {
	// Create a larger graph to test performance
	g := &graph.Graph{
		Nodes: make(map[string]*graph.Node),
		Edges: []*graph.Edge{},
	}

	// Add 20 nodes
	for i := 0; i < 20; i++ {
		nodeID := filepath.Join("aws_instance", "web", string(rune(i)))
		g.Nodes[nodeID] = &graph.Node{
			ID:       nodeID,
			Type:     "aws_instance",
			Name:     string(rune('a' + i)),
			Provider: "aws",
		}
	}

	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "large_diagram.svg")

	ctx := context.Background()
	opts := RenderOptions{
		Format:        "svg",
		Direction:     "TB",
		IncludeLabels: true,
		UseIcons:      false,
	}

	err := RenderDiagram(ctx, g, outputPath, opts)
	if err != nil {
		t.Errorf("RenderDiagram() with large graph error = %v", err)
	}

	// Verify file exists and has substantial content
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Errorf("Failed to read output file: %v", err)
	}
	if len(content) < 1000 {
		t.Error("Large graph SVG should have substantial content")
	}
}

func TestRenderDiagram_InvalidOutputPath(t *testing.T) {
	g := &graph.Graph{
		Nodes: map[string]*graph.Node{
			"aws_instance.web": {
				ID:       "aws_instance.web",
				Type:     "aws_instance",
				Name:     "web",
				Provider: "aws",
			},
		},
		Edges: []*graph.Edge{},
	}

	// Try to write to a directory that doesn't exist and can't be created
	outputPath := "/nonexistent/directory/diagram.svg"

	ctx := context.Background()
	opts := RenderOptions{
		Format:    "svg",
		Direction: "TB",
	}

	err := RenderDiagram(ctx, g, outputPath, opts)
	if err == nil {
		t.Error("RenderDiagram() with invalid output path should return error")
	}
}
