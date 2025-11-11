package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

// BackendConfig represents a Terraform backend configuration
type BackendConfig struct {
	Type       string                 // "local", "remote", "s3", "azurerm", "gcs", "http"
	Config     map[string]interface{} // Backend-specific configuration
	WorkingDir string                 // Directory where terraform files are located
}

// BackendType represents supported backend types
type BackendType string

const (
	BackendTypeLocal    BackendType = "local"
	BackendTypeRemote   BackendType = "remote"
	BackendTypeS3       BackendType = "s3"
	BackendTypeAzureRM  BackendType = "azurerm"
	BackendTypeGCS      BackendType = "gcs"
	BackendTypeHTTP     BackendType = "http"
	BackendTypeConsul   BackendType = "consul"
	BackendTypeEtcdV3   BackendType = "etcdv3"
	BackendTypePg       BackendType = "pg"
)

// ParseBackendConfig reads Terraform configuration files and extracts backend configuration
func ParseBackendConfig(configPath string) (*BackendConfig, error) {
	parser := hclparse.NewParser()

	// Find all .tf files in the directory
	var tfFiles []string
	err := filepath.Walk(configPath, func(path string, info os.FileInfo, err error) error {
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

	// Parse each file looking for terraform blocks
	for _, tfFile := range tfFiles {
		backend, err := parseBackendFromFile(parser, tfFile, configPath)
		if err != nil {
			// Continue looking in other files
			continue
		}
		if backend != nil {
			return backend, nil
		}
	}

	// No backend configuration found - default to local backend
	return &BackendConfig{
		Type:       string(BackendTypeLocal),
		Config:     map[string]interface{}{},
		WorkingDir: configPath,
	}, nil
}

// parseBackendFromFile parses a single .tf file looking for backend configuration
func parseBackendFromFile(parser *hclparse.Parser, path string, workingDir string) (*BackendConfig, error) {
	file, diags := parser.ParseHCLFile(path)
	if diags.HasErrors() {
		return nil, fmt.Errorf("HCL parse errors: %s", diags.Error())
	}

	// Look for terraform blocks
	content, _, diags := file.Body.PartialContent(&hcl.BodySchema{
		Blocks: []hcl.BlockHeaderSchema{
			{
				Type:       "terraform",
				LabelNames: []string{},
			},
		},
	})
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to parse body: %s", diags.Error())
	}

	// Process terraform blocks
	for _, block := range content.Blocks {
		if block.Type != "terraform" {
			continue
		}

		// Look for backend block within terraform block
		backend, err := parseBackendBlock(block.Body, workingDir)
		if err != nil {
			continue
		}
		if backend != nil {
			return backend, nil
		}
	}

	return nil, nil
}

// parseBackendBlock extracts backend configuration from a terraform block
func parseBackendBlock(body hcl.Body, workingDir string) (*BackendConfig, error) {
	content, _, diags := body.PartialContent(&hcl.BodySchema{
		Blocks: []hcl.BlockHeaderSchema{
			{
				Type:       "backend",
				LabelNames: []string{"type"},
			},
		},
	})
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to parse backend: %s", diags.Error())
	}

	for _, block := range content.Blocks {
		if block.Type != "backend" {
			continue
		}

		if len(block.Labels) == 0 {
			continue
		}

		backendType := block.Labels[0]
		config, err := parseBackendAttributes(block.Body)
		if err != nil {
			config = make(map[string]interface{})
		}

		return &BackendConfig{
			Type:       backendType,
			Config:     config,
			WorkingDir: workingDir,
		}, nil
	}

	return nil, nil
}

// parseBackendAttributes extracts attributes from a backend block
func parseBackendAttributes(body hcl.Body) (map[string]interface{}, error) {
	config := make(map[string]interface{})

	// Try to get syntax body for better parsing
	if syntaxBody, ok := body.(*hclsyntax.Body); ok {
		// Parse attributes
		for name, attr := range syntaxBody.Attributes {
			val, diags := attr.Expr.Value(nil)
			if diags.HasErrors() {
				// Try to extract as string literal
				if strVal := extractStringLiteral(attr.Expr); strVal != "" {
					config[name] = strVal
					continue
				}
				continue
			}
			config[name] = ctyToInterface(val)
		}

		// Parse nested blocks (like workspaces)
		for _, block := range syntaxBody.Blocks {
			blockConfig := make(map[string]interface{})
			for name, attr := range block.Body.Attributes {
				val, diags := attr.Expr.Value(nil)
				if diags.HasErrors() {
					continue
				}
				blockConfig[name] = ctyToInterface(val)
			}
			config[block.Type] = blockConfig
		}
	} else {
		// Fallback to basic attribute parsing
		attrs, diags := body.JustAttributes()
		if !diags.HasErrors() {
			for name, attr := range attrs {
				val, diags := attr.Expr.Value(nil)
				if diags.HasErrors() {
					continue
				}
				config[name] = ctyToInterface(val)
			}
		}
	}

	return config, nil
}

// extractStringLiteral attempts to extract a string from an expression
func extractStringLiteral(expr hclsyntax.Expression) string {
	if template, ok := expr.(*hclsyntax.TemplateExpr); ok {
		if len(template.Parts) == 1 {
			if literal, ok := template.Parts[0].(*hclsyntax.LiteralValueExpr); ok {
				if literal.Val.Type() == cty.String {
					return literal.Val.AsString()
				}
			}
		}
	}
	return ""
}

// GetStatePath resolves the state file path based on backend configuration
func GetStatePath(backend *BackendConfig) (string, error) {
	switch BackendType(backend.Type) {
	case BackendTypeLocal:
		return getLocalStatePath(backend)
	case BackendTypeRemote, BackendTypeS3, BackendTypeAzureRM, BackendTypeGCS, BackendTypeHTTP:
		// These require special handling - state is not on local filesystem
		return "", fmt.Errorf("backend type '%s' requires remote state fetching", backend.Type)
	default:
		return "", fmt.Errorf("unsupported backend type: %s", backend.Type)
	}
}

// getLocalStatePath resolves the path for local backend
func getLocalStatePath(backend *BackendConfig) (string, error) {
	// Check if path is specified in backend config
	if path, ok := backend.Config["path"].(string); ok && path != "" {
		// Path is relative to working directory
		fullPath := filepath.Join(backend.WorkingDir, path)
		if _, err := os.Stat(fullPath); err == nil {
			return fullPath, nil
		}
		return "", fmt.Errorf("state file not found at configured path: %s", fullPath)
	}

	// Default local backend path
	defaultPath := filepath.Join(backend.WorkingDir, "terraform.tfstate")
	if _, err := os.Stat(defaultPath); err == nil {
		return defaultPath, nil
	}

	// Try .terraform directory
	terraformPath := filepath.Join(backend.WorkingDir, ".terraform", "terraform.tfstate")
	if _, err := os.Stat(terraformPath); err == nil {
		return terraformPath, nil
	}

	return "", fmt.Errorf("no state file found in working directory: %s", backend.WorkingDir)
}

// AutoDetectStatePath attempts to find the state file without backend configuration
// Tries multiple common locations
func AutoDetectStatePath(configPath string) (string, error) {
	// List of paths to try, in order of preference
	candidates := []string{
		filepath.Join(configPath, "terraform.tfstate"),
		filepath.Join(configPath, ".terraform", "terraform.tfstate"),
		filepath.Join(configPath, "state", "terraform.tfstate"),
		filepath.Join(configPath, "..", "terraform.tfstate"), // Parent directory
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("no state file found in common locations under: %s", configPath)
}
