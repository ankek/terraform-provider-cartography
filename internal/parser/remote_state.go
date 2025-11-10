package parser

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/hashicorp/go-retryablehttp"
)

// RemoteStateConfig holds configuration for fetching remote state
type RemoteStateConfig struct {
	Backend *BackendConfig
	// Authentication credentials (optional overrides - backend config takes priority)
	TerraformToken string // For Terraform Cloud/Enterprise
	AWSAccessKey   string // For S3
	AWSSecretKey   string
	AWSSessionToken string // Optional session token for temporary credentials
	AWSProfile      string // AWS profile name
	AzureAccount    string // For Azure Storage
	AzureKey        string
	GCPCredentials  string // For GCS (JSON key)
}

// getCredentialFromBackendOrEnv gets a credential from backend config, then env var, then fallback
func getCredentialFromBackendOrEnv(backend *BackendConfig, configKey string, envVars []string, fallback string) string {
	// Priority 1: Check backend configuration
	if val, ok := backend.Config[configKey].(string); ok && val != "" {
		return val
	}

	// Priority 2: Check environment variables
	for _, envVar := range envVars {
		if val := os.Getenv(envVar); val != "" {
			return val
		}
	}

	// Priority 3: Use fallback value
	return fallback
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

// fetchS3State retrieves state from AWS S3 using AWS SDK v2
func fetchS3State(ctx context.Context, remoteConfig *RemoteStateConfig) ([]byte, error) {
	backend := remoteConfig.Backend

	bucket, ok := backend.Config["bucket"].(string)
	if !ok || bucket == "" {
		return nil, fmt.Errorf("bucket not specified in S3 backend configuration")
	}

	key, ok := backend.Config["key"].(string)
	if !ok || key == "" {
		return nil, fmt.Errorf("key not specified in S3 backend configuration")
	}

	// Get AWS region from backend config or environment
	region := getCredentialFromBackendOrEnv(backend, "region",
		[]string{"AWS_DEFAULT_REGION", "AWS_REGION"}, "us-east-1")

	// Get AWS credentials with priority: backend config -> provider config -> environment
	var accessKey, secretKey, sessionToken, profile string

	// Check backend configuration first
	accessKey = getCredentialFromBackendOrEnv(backend, "access_key",
		[]string{"AWS_ACCESS_KEY_ID"}, "")
	secretKey = getCredentialFromBackendOrEnv(backend, "secret_key",
		[]string{"AWS_SECRET_ACCESS_KEY"}, "")
	sessionToken = getCredentialFromBackendOrEnv(backend, "token",
		[]string{"AWS_SESSION_TOKEN"}, "")
	profile = getCredentialFromBackendOrEnv(backend, "profile",
		[]string{"AWS_PROFILE"}, "")

	// Override with provider config if provided (but backend config takes priority)
	if accessKey == "" && remoteConfig.AWSAccessKey != "" {
		accessKey = remoteConfig.AWSAccessKey
	}
	if secretKey == "" && remoteConfig.AWSSecretKey != "" {
		secretKey = remoteConfig.AWSSecretKey
	}
	if sessionToken == "" && remoteConfig.AWSSessionToken != "" {
		sessionToken = remoteConfig.AWSSessionToken
	}
	if profile == "" && remoteConfig.AWSProfile != "" {
		profile = remoteConfig.AWSProfile
	}

	// Build AWS config with proper credential chain
	var cfg aws.Config
	var err error

	// Priority 1: Use explicit credentials if provided
	if accessKey != "" && secretKey != "" {
		cfg, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(region),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
				accessKey,
				secretKey,
				sessionToken,
			)),
		)
	} else if profile != "" {
		// Priority 2: Use AWS profile
		cfg, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(region),
			config.WithSharedConfigProfile(profile),
		)
	} else {
		// Priority 3: Use default credential chain (env vars, shared config, IAM role, etc.)
		cfg, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(region),
		)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to load AWS configuration: %w", err)
	}

	// Create S3 client
	client := s3.NewFromConfig(cfg)

	// Get the object from S3
	result, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch state from S3 (bucket=%s, key=%s, region=%s): %w\n"+
			"Hint: Ensure AWS credentials are configured via:\n"+
			"  1. Provider config (aws_access_key, aws_secret_key)\n"+
			"  2. Environment variables (AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY)\n"+
			"  3. AWS shared credentials file (~/.aws/credentials)\n"+
			"  4. IAM role (if running on EC2, ECS, Lambda, etc.)",
			bucket, key, region, err)
	}
	defer result.Body.Close()

	// Read the state data
	data, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read S3 state data: %w", err)
	}

	return data, nil
}

// fetchAzureState retrieves state from Azure Blob Storage using Azure SDK
func fetchAzureState(ctx context.Context, remoteConfig *RemoteStateConfig) ([]byte, error) {
	backend := remoteConfig.Backend

	storageAccount, ok := backend.Config["storage_account_name"].(string)
	if !ok || storageAccount == "" {
		return nil, fmt.Errorf("storage_account_name not specified in azurerm backend configuration")
	}

	containerName, ok := backend.Config["container_name"].(string)
	if !ok || containerName == "" {
		return nil, fmt.Errorf("container_name not specified in azurerm backend configuration")
	}

	key, ok := backend.Config["key"].(string)
	if !ok || key == "" {
		return nil, fmt.Errorf("key not specified in azurerm backend configuration")
	}

	// Get credentials with priority: backend config -> provider config -> environment
	accountKey := getCredentialFromBackendOrEnv(backend, "access_key",
		[]string{"ARM_ACCESS_KEY", "AZURE_STORAGE_KEY"}, "")

	// Override with provider config if provided (but backend config takes priority)
	if accountKey == "" && remoteConfig.AzureKey != "" {
		accountKey = remoteConfig.AzureKey
	}

	if accountKey == "" {
		return nil, fmt.Errorf("Azure Storage account key not found. Set one of:\n"+
			"  1. Backend config: access_key in azurerm backend block\n"+
			"  2. Environment variable: ARM_ACCESS_KEY\n"+
			"  3. Environment variable: AZURE_STORAGE_KEY\n"+
			"  4. Provider config: azure_key (optional)")
	}

	// Create credential from account key
	credential, err := azblob.NewSharedKeyCredential(storageAccount, accountKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure credentials: %w", err)
	}

	// Create blob client
	client, err := azblob.NewClientWithSharedKeyCredential(
		fmt.Sprintf("https://%s.blob.core.windows.net/", storageAccount),
		credential,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure blob client: %w", err)
	}

	// Download the blob
	downloadResponse, err := client.DownloadStream(ctx, containerName, key, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if ok := errors.As(err, &respErr); ok {
			if respErr.StatusCode == 404 {
				return nil, fmt.Errorf("state file not found in Azure Storage (account=%s, container=%s, key=%s)",
					storageAccount, containerName, key)
			}
			if respErr.StatusCode == 403 {
				return nil, fmt.Errorf("access denied to Azure Storage. Verify:\n"+
					"  - Storage account name is correct\n"+
					"  - Account key is valid\n"+
					"  - Container exists and is accessible\n"+
					"  (account=%s, container=%s, key=%s)",
					storageAccount, containerName, key)
			}
		}
		return nil, fmt.Errorf("failed to download from Azure Storage: %w", err)
	}
	defer downloadResponse.Body.Close()

	// Read the state data
	data, err := io.ReadAll(downloadResponse.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read Azure blob data: %w", err)
	}

	return data, nil
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

	// Try fetching with anonymous/public access
	gcsURL := fmt.Sprintf("https://storage.googleapis.com/%s/%s", bucket, prefix)

	client := retryablehttp.NewClient()
	client.RetryMax = 3
	client.Logger = nil

	req, err := retryablehttp.NewRequestWithContext(ctx, "GET", gcsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCS request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from GCS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 403 || resp.StatusCode == 401 {
		return nil, fmt.Errorf("GCS bucket requires authentication. This provider currently supports:\n"+
			"  1. Public GCS buckets (no credentials needed)\n"+
			"  2. Terraform Cloud backend (use terraform_token)\n"+
			"\nFor private GCS buckets, please:\n"+
			"  - Make the state file publicly readable, OR\n"+
			"  - Use Terraform Cloud backend instead, OR\n"+
			"  - Export state locally: terraform state pull > terraform.tfstate")
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GCS returned HTTP %d for bucket=%s, prefix=%s",
			resp.StatusCode, bucket, prefix)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read GCS response: %w", err)
	}

	return data, nil
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
		return ParseStateFile(ctx, statePath)
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
