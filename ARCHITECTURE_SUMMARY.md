# Terraform Provider Cartography - Architecture Summary

## System Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────┐
│                        Terraform Configuration                      │
│                    (User's .tf files and state)                     │
└────────────────────┬────────────────────────────────────────────────┘
                     │
┌────────────────────┴────────────────────────────────────────────────┐
│                     Provider Configuration Layer                    │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ Provider (CartographyProvider)                                │  │
│  │  - terraform_token (TFE)                                      │  │
│  │  - aws_access_key, aws_secret_key (S3)                        │  │
│  │  - azure_account, azure_key (Azure)                           │  │
│  │  - gcp_credentials (GCS)                                      │  │
│  └────────┬──────────────────┬───────────────────────────────────┘  │
│           │                  │                                      │
│  ┌────────▼────┐  ┌──────────▼──────────┐                           │
│  │ Data Source │  │ Resource            │                           │
│  │ (Read-Only) │  │ (Full CRUD)         │                           │
│  └────────┬────┘  └──────────┬──────────┘                           │
└───────────┼──────────────────┼──────────────────────────────────────┘
            │                  │
            └──────────┬───────┘
                       │
        ┌──────────────▼──────────────┐
        │ DiagramGenerator.Generate() │
        └──────────────┬──────────────┘
                       │
    ┌──────────────────┼──────────────────┐
    │                  │                  │
    ▼                  ▼                  ▼
┌─────────────┐  ┌──────────────┐  ┌──────────────┐
│ Validation  │  │ State Loader │  │ Parser       │
│ (path.go)   │  │ (state_      │  │ (parser/)    │
│             │  │  loader.go)  │  │              │
└─────────────┘  └──────────────┘  └──────────────┘
                       │
        ┌──────────────┼──────────────┐
        │              │              │
        ▼              ▼              ▼
    ┌────────┐  ┌──────────┐  ┌──────────────┐
    │ Local  │  │ Backend  │  │ HCL Parser   │
    │ Files  │  │ Parser   │  │ (hcl_        │
    │        │  │ (backend │  │  parser.go)  │
    └────────┘  │  _parser │  └──────────────┘
                │  .go)    │
                └────┬─────┘
                     │
    ┌────────────────┼───────────────────┐
    │                │                   │
    ▼                ▼                   ▼
┌─────────┐  ┌──────────────┐  ┌────────────────────┐
│ParseState│  │Fetch Remote  │  │Auto-Detect State   │
│File      │  │State (remote │  │Path                │
│(state_   │  │_state.go)    │  │(auto-detect logic) │
│parser.go)│  │              │  │                    │
└─────────┘  └──────────────┘  └────────────────────┘
    │                │
    │         ┌──────┴───────┬──────────┬────────┐
    │         │              │          │        │
    │         ▼              ▼          ▼        ▼
    │    ┌────────┐    ┌──────┐   ┌──────┐  ┌──────┐
    │    │ TFE    │    │ S3   │   │Azure │  │ GCS  │
    │    │ Cloud  │    │      │   │ Blob │  │      │
    │    └────────┘    └──────┘   └──────┘  └──────┘
    │         │              │          │        │
    └─────────┴──────────────┴──────────┴────────┘
              │
              ▼
    ┌──────────────────┐
    │ Parsed Resources │
    │ (Resource[])     │
    └────────┬─────────┘
             │
    ┌────────▼──────────┐
    │ Graph Builder     │
    │ (graph.go)        │
    └────────┬──────────┘
             │
    ┌────────▼──────────┐
    │ Resource Graph    │
    │ (with deps)       │
    └────────┬──────────┘
             │
    ┌────────▼──────────┐
    │ Renderer          │
    │ (renderer.go)     │
    └────────┬──────────┘
             │
    ┌────────┴──────────────┬──────────┐
    │                       │          │
    ▼                       ▼          ▼
 ┌───────┐            ┌───────────┐  ┌──────┐
 │ SVG   │            │ PNG/JPEG  │  │Layout│
 │Render │            │(resvg,    │  │      │
 │       │            │ inkscape) │  │      │
 └───────┘            └───────────┘  └──────┘
    │                       │
    └───────────┬───────────┘
                │
        ┌───────▼──────┐
        │ Output File  │
        │ (.svg/.png)  │
        └──────────────┘
```

## Data Flow: Request to Diagram

### Scenario 1: Explicit State Path

```
User specifies: state_path = "./terraform.tfstate"
    │
    ▼
state_loader.LoadResources()
    │
    ▼ (Priority 1)
Check state_path
    │
    ▼ (Found)
state_parser.ParseStateFile(state_path)
    │
    ├─ Read JSON file
    ├─ Detect version (v3 vs v4+)
    ├─ Filter managed resources
    ├─ Extract provider from type
    └─ Build Resource[]
    │
    ▼ (Resources ready)
Continue to graph building...
```

### Scenario 2: Config Path with Backend Detection

```
User specifies: config_path = "./terraform"
    │
    ▼
state_loader.LoadResources()
    │
    ▼ (Priority 2)
backend_parser.ParseBackendConfig(config_path)
    │
    ├─ Walk directory for .tf files
    ├─ Parse HCL for terraform blocks
    ├─ Extract backend block
    └─ Return BackendConfig
    │
    ▼ (Backend found)
loadFromBackend(providerConfig, backend)
    │
    ├─ If local: GetStatePath() → ParseStateFile()
    │
    └─ If remote:
        │
        ├─ Build RemoteStateConfig with credentials
        │
        ▼
        remote_state.FetchRemoteState()
        │
        ├─ Dispatch to backend-specific handler
        │
        ├─ For S3: Fetch from bucket via HTTPS
        ├─ For Azure: Fetch from blob via HTTPS
        ├─ For TFE: API call to get state
        └─ For GCS: Fetch from bucket via HTTPS
        │
        ▼
        Parse downloaded JSON state
```

### Scenario 3: Auto-Detect (No Paths Specified)

```
No state_path or config_path provided
    │
    ▼
state_loader.LoadResources()
    │
    ▼ (Priority 3)
Auto-detect in current directory "."
    │
    ├─ Try backend detection
    │  ├─ ParseBackendConfig(".")
    │  └─ If found: loadFromBackend()
    │
    ├─ If not found, try state file auto-detection
    │  ├─ Try: ./terraform.tfstate
    │  ├─ Try: ./.terraform/terraform.tfstate
    │  ├─ Try: ./state/terraform.tfstate
    │  └─ Try: ../terraform.tfstate
    │
    └─ Last resort: ParseConfigDirectory(".")
       └─ Parse all .tf files in current directory
```

## Credential Flow for Remote Backends

```
Provider Configuration
    │
    ├─ terraform_token
    ├─ aws_access_key
    ├─ aws_secret_key
    ├─ azure_account
    ├─ azure_key
    └─ gcp_credentials
    │
    ▼
CartographyProviderModel
    │
    ▼ (passed to data source/resource)
DiagramDataSource.Read() / DiagramResource.Create()
    │
    ▼
DiagramGenerator.Generate()
    │
    ▼
LoadResources(ctx, providerConfig, ...)
    │
    ├─ Check state_path (no credentials needed)
    │
    └─ Check config_path → backend detection
        │
        ▼
        loadFromBackend(providerConfig, backend)
        │
        ▼ (if remote)
        Build RemoteStateConfig
        │
        ├─ remoteConfig.TerraformToken = providerConfig.TerraformToken
        ├─ remoteConfig.AWSAccessKey = providerConfig.AWSAccessKey
        ├─ remoteConfig.AWSSecretKey = providerConfig.AWSSecretKey
        ├─ remoteConfig.AzureAccount = providerConfig.AzureAccount
        ├─ remoteConfig.AzureKey = providerConfig.AzureKey
        └─ remoteConfig.GCPCredentials = providerConfig.GCPCredentials
        │
        ▼
        Fallback to environment variables if not set
        (TFE_TOKEN, AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, etc.)
        │
        ▼
        FetchRemoteState(remoteConfig)
```

## State Format Support

```
Terraform State File (terraform.tfstate)
    │
    ▼
JSON Parsing
    │
    ├─ Check Values (root_module present?)
    │  │
    │  ├─ YES → Modern Format (v4+)
    │  │  └─ Use: state.Values.RootModule.Resources
    │  │
    │  └─ NO → Legacy Format (v3)
    │     └─ Use: state.Resources
    │
    ▼
Extract Resources
    │
    ├─ Filter: mode == "managed" only
    ├─ For each instance:
    │  ├─ Extract Type, Name, Attributes
    │  ├─ Determine Provider (aws, azure, gcp, etc.)
    │  ├─ Build ID (type.name or type.name[index])
    │  └─ Capture Dependencies
    │
    ▼
Resource[] (Internal representation)
```

## Key Integration Points

### 1. Provider → Data Source/Resource
- Provider stores credentials
- Passes to DiagramDataSource.Configure() / DiagramResource.Configure()
- (Currently unused but infrastructure is ready)

### 2. Generator → State Loader
- Generator calls parseResources()
- parseResources() uses LoadResources() for smart loading
- Returns consistent Resource[] type

### 3. State Loader → Parser
- Dispatches to appropriate parser:
  - ParseStateFile() for explicit paths
  - ParseBackendConfig() for backend detection
  - ParseConfigDirectory() for HCL parsing

### 4. Remote State → HTTP Client
- Uses retryablehttp for resilient requests
- 3-attempt retry strategy
- Handles both authenticated (TFE) and unauthenticated (public buckets) access

### 5. Parser → Resource Transformation
- StateResource → Resource
- Preserves dependencies
- Categorizes by resource type
- Filters non-infrastructure resources

### 6. Graph Builder → Renderer
- Takes Resource[] with dependencies
- Builds graph structure
- Layout algorithm positions nodes
- Renderer creates visual output

---

## Implementation Patterns

### Pattern 1: Graceful Degradation
```
Primary Method (Fast)
    ↓ (if fails)
Fallback 1 (Slower)
    ↓ (if fails)
Fallback 2 (Slowest)
    ↓ (if fails)
Error
```

Example: state_path → backend detection → auto-detect → HCL parse → error

### Pattern 2: Configuration Dispatch
```
Backend Type (from config)
    │
    ├─ local → Local file handler
    ├─ remote → TFE API handler
    ├─ s3 → S3 HTTP handler
    ├─ azurerm → Azure HTTP handler
    ├─ gcs → GCS HTTP handler
    └─ http → Generic HTTP handler
```

### Pattern 3: Shared Generator
```
DiagramDataSource
    │
    └─ Uses DiagramGenerator
    
DiagramResource
    │
    └─ Uses DiagramGenerator

Benefit: Single source of truth for diagram generation logic
```

---

## Extensibility Points

### For Adding New Backends:

1. **Backend Parser** (`backend_parser.go`)
   - Add backend type constant
   - Extend `ParseBackendAttributes()` if new config keys needed

2. **Remote State** (`remote_state.go`)
   - Add `fetch{BackendType}State()` function
   - Add case to `FetchRemoteState()` dispatcher
   - Add credential fields to `RemoteStateConfig` if needed

3. **Provider** (`provider.go`)
   - Add credential fields if required
   - Pass credentials through state_loader.go

4. **Tests**
   - Add test cases for new backend
   - Test credential handling
   - Test error scenarios

### For Adding New Resource Types:

1. **Types** (`types.go`)
   - Add resource type to appropriate map in `GetResourceType()`
   - Add filtering rules if needed in `IsCloudInfraResource()`

2. **Icons** (optional)
   - Add icon for new resource type in `icons.go`

3. **Rendering** (optional)
   - Add color scheme in `colors.go` if custom coloring needed

---

## Module Dependencies

```
provider/
    ├─ imports: parser/, validation/
    └─ imports: graph/, renderer/

parser/
    ├─ imports: (stdlib only)
    ├─ hashicorp/hcl
    ├─ go-retryablehttp
    └─ go-cty

graph/
    ├─ imports: parser/
    └─ (stdlib only)

renderer/
    ├─ imports: graph/
    └─ (stdlib + image/svg+xml, external tools)

validation/
    └─ imports: (stdlib only)
```

---

This summary provides a complete architectural overview for understanding and extending the Cartography provider.
