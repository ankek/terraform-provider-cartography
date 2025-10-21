package parser

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

// ParseConfigDirectory reads and parses all .tf files in a directory.
// It respects the provided context for cancellation.
func ParseConfigDirectory(ctx context.Context, dirPath string) ([]Resource, error) {
	// Check if context is already cancelled
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	parser := hclparse.NewParser()

	// Find all .tf files
	var tfFiles []string
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".tf") {
			tfFiles = append(tfFiles, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to scan directory: %w", err)
	}

	var resources []Resource
	for _, tfFile := range tfFiles {
		fileResources, err := parseHCLFile(parser, tfFile)
		if err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", tfFile, err)
		}
		resources = append(resources, fileResources...)
	}

	return resources, nil
}

// parseHCLFile parses a single HCL file and extracts resources
func parseHCLFile(parser *hclparse.Parser, path string) ([]Resource, error) {
	file, diags := parser.ParseHCLFile(path)
	if diags.HasErrors() {
		return nil, fmt.Errorf("HCL parse errors: %s", diags.Error())
	}

	var resources []Resource

	// Parse the file body
	content, _, diags := file.Body.PartialContent(&hcl.BodySchema{
		Blocks: []hcl.BlockHeaderSchema{
			{
				Type:       "resource",
				LabelNames: []string{"type", "name"},
			},
		},
	})
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to parse body: %s", diags.Error())
	}

	// Extract resources
	for _, block := range content.Blocks {
		if block.Type != "resource" {
			continue
		}

		resourceType := block.Labels[0]
		resourceName := block.Labels[1]
		provider := extractProvider(resourceType)

		// Parse resource attributes
		attrs, err := parseResourceAttributes(block.Body)
		if err != nil {
			// Log warning but continue
			attrs = make(map[string]interface{})
		}

		// Extract dependencies from the block body (traversals)
		deps := extractDependenciesFromBlock(block.Body)

		resource := Resource{
			Type:         resourceType,
			Name:         resourceName,
			Provider:     provider,
			Attributes:   attrs,
			ID:           fmt.Sprintf("%s.%s", resourceType, resourceName),
			Dependencies: deps,
		}

		resources = append(resources, resource)
	}

	return resources, nil
}

// parseResourceAttributes extracts attributes from a resource block
func parseResourceAttributes(body hcl.Body) (map[string]interface{}, error) {
	attrs := make(map[string]interface{})

	// Get all attributes
	hclAttrs, diags := body.JustAttributes()
	if diags.HasErrors() {
		return attrs, fmt.Errorf("failed to parse attributes: %s", diags.Error())
	}

	for name, attr := range hclAttrs {
		val, diags := attr.Expr.Value(nil)
		if diags.HasErrors() {
			// Skip attributes that can't be evaluated without context
			continue
		}

		attrs[name] = ctyToInterface(val)
	}

	return attrs, nil
}

// ctyToInterface converts a cty.Value to a native Go interface
func ctyToInterface(val cty.Value) interface{} {
	if val.IsNull() {
		return nil
	}

	switch val.Type() {
	case cty.String:
		return val.AsString()
	case cty.Number:
		f, _ := val.AsBigFloat().Float64()
		return f
	case cty.Bool:
		return val.True()
	}

	if val.Type().IsListType() || val.Type().IsTupleType() {
		var list []interface{}
		it := val.ElementIterator()
		for it.Next() {
			_, v := it.Element()
			list = append(list, ctyToInterface(v))
		}
		return list
	}

	if val.Type().IsMapType() || val.Type().IsObjectType() {
		m := make(map[string]interface{})
		it := val.ElementIterator()
		for it.Next() {
			k, v := it.Element()
			m[k.AsString()] = ctyToInterface(v)
		}
		return m
	}

	return nil
}

// extractDependencies finds resource references in attributes
func extractDependencies(attrs map[string]interface{}) []string {
	var deps []string

	for _, val := range attrs {
		switch v := val.(type) {
		case string:
			// Look for references like "azurerm_virtual_network.main.id"
			if strings.Contains(v, ".") && !strings.HasPrefix(v, "var.") {
				parts := strings.Split(v, ".")
				if len(parts) >= 2 {
					dep := fmt.Sprintf("%s.%s", parts[0], parts[1])
					deps = append(deps, dep)
				}
			}
		case []interface{}:
			for _, item := range v {
				if strItem, ok := item.(string); ok {
					if strings.Contains(strItem, ".") && !strings.HasPrefix(strItem, "var.") {
						parts := strings.Split(strItem, ".")
						if len(parts) >= 2 {
							dep := fmt.Sprintf("%s.%s", parts[0], parts[1])
							deps = append(deps, dep)
						}
					}
				}
			}
		}
	}

	return deps
}

// extractDependenciesFromBlock walks the HCL syntax tree to find resource references
func extractDependenciesFromBlock(body hcl.Body) []string {
	deps := make(map[string]bool) // Use map to deduplicate

	// Try to get the syntax body for traversal extraction
	if syntaxBody, ok := body.(*hclsyntax.Body); ok {
		extractTraversals(syntaxBody, deps)
	}

	// Convert map to slice
	var result []string
	for dep := range deps {
		result = append(result, dep)
	}

	return result
}

// extractTraversals recursively walks the HCL syntax tree to find all resource references
func extractTraversals(body *hclsyntax.Body, deps map[string]bool) {
	// Check all attributes
	for _, attr := range body.Attributes {
		findTraversalsInExpr(attr.Expr, deps)
	}

	// Check all blocks recursively
	for _, block := range body.Blocks {
		extractTraversals(block.Body, deps)
	}
}

// findTraversalsInExpr finds resource references in an HCL expression
func findTraversalsInExpr(expr hclsyntax.Expression, deps map[string]bool) {
	// Check if this expression is a scope traversal (e.g., digitalocean_vpc.example.id)
	if traversal, ok := expr.(*hclsyntax.ScopeTraversalExpr); ok {
		if len(traversal.Traversal) >= 2 {
			rootName := traversal.Traversal.RootName()

			// Skip variables, locals, data sources, etc. - only track resource references
			if rootName == "var" || rootName == "local" || rootName == "data" ||
			   rootName == "module" || rootName == "path" || rootName == "terraform" {
				return
			}

			// Get the first two parts: resource_type.resource_name
			if attr, ok := traversal.Traversal[1].(hcl.TraverseAttr); ok {
				dep := fmt.Sprintf("%s.%s", rootName, attr.Name)
				deps[dep] = true
			}
		}
		return
	}

	// Recursively search in composite expressions
	switch e := expr.(type) {
	case *hclsyntax.TupleConsExpr:
		// Handle lists [item1, item2]
		for _, item := range e.Exprs {
			findTraversalsInExpr(item, deps)
		}
	case *hclsyntax.ObjectConsExpr:
		// Handle objects {key = value}
		for _, item := range e.Items {
			findTraversalsInExpr(item.KeyExpr, deps)
			findTraversalsInExpr(item.ValueExpr, deps)
		}
	case *hclsyntax.FunctionCallExpr:
		// Handle function calls like concat(list1, list2)
		for _, arg := range e.Args {
			findTraversalsInExpr(arg, deps)
		}
	case *hclsyntax.ConditionalExpr:
		// Handle ternary expressions condition ? true_val : false_val
		findTraversalsInExpr(e.Condition, deps)
		findTraversalsInExpr(e.TrueResult, deps)
		findTraversalsInExpr(e.FalseResult, deps)
	case *hclsyntax.ForExpr:
		// Handle for expressions
		findTraversalsInExpr(e.CollExpr, deps)
		if e.KeyExpr != nil {
			findTraversalsInExpr(e.KeyExpr, deps)
		}
		findTraversalsInExpr(e.ValExpr, deps)
	case *hclsyntax.IndexExpr:
		// Handle indexing expressions like list[0]
		findTraversalsInExpr(e.Collection, deps)
		findTraversalsInExpr(e.Key, deps)
	case *hclsyntax.BinaryOpExpr:
		// Handle binary operations like a + b
		findTraversalsInExpr(e.LHS, deps)
		findTraversalsInExpr(e.RHS, deps)
	case *hclsyntax.UnaryOpExpr:
		// Handle unary operations like !value
		findTraversalsInExpr(e.Val, deps)
	case *hclsyntax.ParenthesesExpr:
		// Handle parenthesized expressions
		findTraversalsInExpr(e.Expression, deps)
	}
}
