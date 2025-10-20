package parser

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/hashicorp/go-retryablehttp"
)

// RemoteStateConfig holds configuration for fetching remote state
type RemoteStateConfig struct {
	Backend *BackendConfig
	// Authentication credentials
	TerraformToken string // For Terraform Cloud/Enterprise
	AWSAccessKey   string // For S3
	AWSSecretKey   string
	AzureAccount   string // For Azure Storage
	AzureKey       string
	GCPCredentials string // For GCS (JSON key)
}

// FetchRemoteState retrieves state from a remote backend
func FetchRemoteState(ctx context.Context, config *RemoteStateConfig) ([]byte, error) {
	switch BackendType(config.Backend.Type) {
	case BackendTypeRemote:
		return fetchTerraformCloudState(ctx, config)
	case BackendTypeS3:
		return fetchS3State(ctx, config)
	case BackendTypeAzureRM:
		return fetchAzureState(ctx, config)
	case BackendTypeGCS:
		return fetchGCSState(ctx, config)
	case BackendTypeHTTP:
		return fetchHTTPState(ctx, config)
	default:
		return nil, fmt.Errorf("remote state fetching not supported for backend type: %s", config.Backend.Type)
	}
}

// fetchTerraformCloudState retrieves state from Terraform Cloud/Enterprise
func fetchTerraformCloudState(ctx context.Context, config *RemoteStateConfig) ([]byte, error) {
	// Get organization and workspace
	organization, ok := config.Backend.Config["organization"].(string)
	if !ok || organization == "" {
		return nil, fmt.Errorf("organization not specified in remote backend configuration")
	}

	workspaceName := ""
	if workspaces, ok := config.Backend.Config["workspaces"].(map[string]interface{}); ok {
		if name, ok := workspaces["name"].(string); ok {
			workspaceName = name
		}
	}
	if workspaceName == "" {
		return nil, fmt.Errorf("workspace name not specified in remote backend configuration")
	}

	// Get token - prefer config, fall back to environment
	token := config.TerraformToken
	if token == "" {
		token = os.Getenv("TFE_TOKEN")
	}
	if token == "" {
		token = os.Getenv("TF_TOKEN_" + strings.ReplaceAll(organization, "-", "_"))
	}
	if token == "" {
		return nil, fmt.Errorf("Terraform Cloud token not found. Set TFE_TOKEN environment variable or provider configuration")
	}

	// Determine hostname (default to app.terraform.io)
	hostname := "app.terraform.io"
	if h, ok := config.Backend.Config["hostname"].(string); ok && h != "" {
		hostname = h
	}

	// Construct API URL to get workspace
	workspaceURL := fmt.Sprintf("https://%s/api/v2/organizations/%s/workspaces/%s",
		hostname, organization, workspaceName)

	// Fetch workspace details to get current state version
	client := retryablehttp.NewClient()
	client.RetryMax = 3
	client.Logger = nil // Disable logging

	req, err := retryablehttp.NewRequestWithContext(ctx, "GET", workspaceURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create workspace request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/vnd.api+json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch workspace details: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch workspace (status %d): %s", resp.StatusCode, string(body))
	}

	var workspaceResp struct {
		Data struct {
			Relationships struct {
				CurrentStateVersion struct {
					Data struct {
						ID string `json:"id"`
					} `json:"data"`
				} `json:"current-state-version"`
			} `json:"relationships"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&workspaceResp); err != nil {
		return nil, fmt.Errorf("failed to decode workspace response: %w", err)
	}

	stateVersionID := workspaceResp.Data.Relationships.CurrentStateVersion.Data.ID
	if stateVersionID == "" {
		return nil, fmt.Errorf("no current state version found for workspace")
	}

	// Fetch the actual state file
	stateURL := fmt.Sprintf("https://%s/api/v2/state-versions/%s/download",
		hostname, stateVersionID)

	req, err = retryablehttp.NewRequestWithContext(ctx, "GET", stateURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create state request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err = client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch state: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch state (status %d): %s", resp.StatusCode, string(body))
	}

	return io.ReadAll(resp.Body)
}

// fetchS3State retrieves state from AWS S3
func fetchS3State(ctx context.Context, config *RemoteStateConfig) ([]byte, error) {
	bucket, ok := config.Backend.Config["bucket"].(string)
	if !ok || bucket == "" {
		return nil, fmt.Errorf("bucket not specified in S3 backend configuration")
	}

	key, ok := config.Backend.Config["key"].(string)
	if !ok || key == "" {
		return nil, fmt.Errorf("key not specified in S3 backend configuration")
	}

	// Get AWS region
	region := "us-east-1"
	if r, ok := config.Backend.Config["region"].(string); ok && r != "" {
		region = r
	}
	_ = region // TODO: use region when implementing AWS SDK

	// Get credentials - prefer config, fall back to environment
	accessKey := config.AWSAccessKey
	secretKey := config.AWSSecretKey
	if accessKey == "" {
		accessKey = os.Getenv("AWS_ACCESS_KEY_ID")
	}
	if secretKey == "" {
		secretKey = os.Getenv("AWS_SECRET_ACCESS_KEY")
	}

	if accessKey == "" || secretKey == "" {
		return nil, fmt.Errorf("AWS credentials not found. Set AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY")
	}

	// Note: This is a simplified implementation
	// In production, you should use the official AWS SDK
	return nil, fmt.Errorf("S3 backend state fetching requires AWS SDK integration (TODO: implement with github.com/aws/aws-sdk-go-v2)")
}

// fetchAzureState retrieves state from Azure Blob Storage
func fetchAzureState(ctx context.Context, config *RemoteStateConfig) ([]byte, error) {
	storageAccount, ok := config.Backend.Config["storage_account_name"].(string)
	if !ok || storageAccount == "" {
		return nil, fmt.Errorf("storage_account_name not specified in azurerm backend configuration")
	}

	containerName, ok := config.Backend.Config["container_name"].(string)
	if !ok || containerName == "" {
		return nil, fmt.Errorf("container_name not specified in azurerm backend configuration")
	}

	key, ok := config.Backend.Config["key"].(string)
	if !ok || key == "" {
		return nil, fmt.Errorf("key not specified in azurerm backend configuration")
	}

	// Get credentials
	accountKey := config.AzureKey
	if accountKey == "" {
		accountKey = os.Getenv("ARM_ACCESS_KEY")
	}

	if accountKey == "" {
		return nil, fmt.Errorf("Azure Storage account key not found. Set ARM_ACCESS_KEY")
	}

	// Note: This is a simplified implementation
	// In production, you should use the official Azure SDK
	return nil, fmt.Errorf("Azure backend state fetching requires Azure SDK integration (TODO: implement with github.com/Azure/azure-sdk-for-go)")
}

// fetchGCSState retrieves state from Google Cloud Storage
func fetchGCSState(ctx context.Context, config *RemoteStateConfig) ([]byte, error) {
	bucket, ok := config.Backend.Config["bucket"].(string)
	if !ok || bucket == "" {
		return nil, fmt.Errorf("bucket not specified in GCS backend configuration")
	}

	prefix := "default.tfstate"
	if p, ok := config.Backend.Config["prefix"].(string); ok && p != "" {
		prefix = p + "/default.tfstate"
	}
	_ = prefix // TODO: use prefix when implementing GCP SDK

	// Get credentials
	credentials := config.GCPCredentials
	if credentials == "" {
		credentials = os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	}

	if credentials == "" {
		return nil, fmt.Errorf("GCP credentials not found. Set GOOGLE_APPLICATION_CREDENTIALS")
	}

	// Note: This is a simplified implementation
	// In production, you should use the official GCP SDK
	return nil, fmt.Errorf("GCS backend state fetching requires GCP SDK integration (TODO: implement with cloud.google.com/go/storage)")
}

// fetchHTTPState retrieves state from HTTP/HTTPS endpoint
func fetchHTTPState(ctx context.Context, config *RemoteStateConfig) ([]byte, error) {
	address, ok := config.Backend.Config["address"].(string)
	if !ok || address == "" {
		return nil, fmt.Errorf("address not specified in HTTP backend configuration")
	}

	client := retryablehttp.NewClient()
	client.RetryMax = 3
	client.Logger = nil

	req, err := retryablehttp.NewRequestWithContext(ctx, "GET", address, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Add optional authentication
	if username, ok := config.Backend.Config["username"].(string); ok && username != "" {
		if password, ok := config.Backend.Config["password"].(string); ok && password != "" {
			req.SetBasicAuth(username, password)
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch state from HTTP backend: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch state (status %d): %s", resp.StatusCode, string(body))
	}

	return io.ReadAll(resp.Body)
}

// LoadStateFromBackend is a high-level function that handles all backend types
func LoadStateFromBackend(ctx context.Context, config *RemoteStateConfig) ([]Resource, error) {
	// For local backend, use file-based parsing
	if BackendType(config.Backend.Type) == BackendTypeLocal {
		statePath, err := GetStatePath(config.Backend)
		if err != nil {
			return nil, err
		}
		return ParseStateFile(statePath)
	}

	// For remote backends, fetch state and parse
	stateData, err := FetchRemoteState(ctx, config)
	if err != nil {
		return nil, err
	}

	// Parse the state data
	var state TerraformState
	if err := json.Unmarshal(stateData, &state); err != nil {
		return nil, fmt.Errorf("failed to parse remote state: %w", err)
	}

	// Extract resources (same logic as ParseStateFile)
	var stateResources []StateResource
	if state.Values != nil && state.Values.RootModule != nil {
		stateResources = state.Values.RootModule.Resources
	} else {
		stateResources = state.Resources
	}

	var resources []Resource
	for _, stateRes := range stateResources {
		if stateRes.Mode != "managed" {
			continue
		}

		provider := extractProvider(stateRes.Type)

		for idx, instance := range stateRes.Instances {
			var resourceID string
			if len(stateRes.Instances) == 1 {
				resourceID = fmt.Sprintf("%s.%s", stateRes.Type, stateRes.Name)
			} else {
				resourceID = fmt.Sprintf("%s.%s[%d]", stateRes.Type, stateRes.Name, idx)
			}

			resource := Resource{
				Type:         stateRes.Type,
				Name:         stateRes.Name,
				Provider:     provider,
				Attributes:   instance.Attributes,
				ID:           resourceID,
				Dependencies: instance.Dependencies,
			}

			resources = append(resources, resource)
		}
	}

	return resources, nil
}
