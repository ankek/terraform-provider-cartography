<a href="https://terraform.io">
    <img src=".github/tf.png" alt="Terraform logo" title="Terraform" align="left" height="50" />
</a>

# Terraform Provider for Infrastructure Cartography

The Cartography Provider allows you to **automatically generate visual diagrams** of your Terraform infrastructure. It reads your Terraform state or configuration files and creates beautiful diagrams showing your resources and their relationships.

---

## Features

- 🗺️ **Automatic Diagram Generation** - Convert Terraform state into visual architecture diagrams
- 🎨 **Multiple Output Formats** - Export as SVG
- ☁️ **Multi-Cloud Support** - Works with AWS, Azure, Google Cloud, DigitalOcean and other Cloud providers
- 🔗 **Relationship Mapping** - Automatically detects and visualizes dependencies between resources
- 🎯 **Icon Support** - Uses cloud provider icons for diagrams
- 📐 **Flexible Layouts** - Top-to-bottom, left-to-right, and more layout options
- 🏷️ **Smart Labeling** - Includes resource names, types, and key attributes
- 🔄 **Remote State Support** - Works with S3, Azure Blob, GCS, and Terraform Cloud backends

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.25 (for building from source)

## Using the Provider

```hcl
terraform {
  required_providers {
    cartography = {
      source  = "ankek/cartography"
      version = "~> 0.1"
    }
  }
}

provider "cartography" {
  # Optional: Configure remote state backend access
  # aws_access_key = "..."  # Or set AWS_ACCESS_KEY_ID
  # azure_account  = "..."
  # gcp_credentials = file("credentials.json")
}

# Generate a diagram from your infrastructure
resource "cartography_diagram" "main" {
  output_path    = "${path.module}/infrastructure.svg"
  format         = "svg"
  direction      = "TB"
  include_labels = true
  use_icons      = true
  title          = "My Infrastructure"

  # Read from local state file
  state_path = "${path.module}/terraform.tfstate"

  # Or read from configuration files
  # config_path = "${path.module}"
}

# Output the number of resources visualized
output "resource_count" {
  value = cartography_diagram.main.id
}
```

### Using as a Data Source

```hcl
# Generate diagram without managing it as a resource
data "cartography_diagram" "readonly" {
  output_path    = "${path.module}/infra-diagram.png"
  format         = "png"
  state_path     = "${path.module}/terraform.tfstate"
  title          = "Production Infrastructure"
  use_icons      = true
}

output "total_resources" {
  value = data.cartography_diagram.readonly.resource_count
}
```

## Documentation

- [Provider Documentation](https://registry.terraform.io/providers/ankek/cartography/latest/docs) on the Terraform Registry
- [Usage Examples](./examples) in this repository
- [Resource Schema](./docs/resources/diagram.md) - Detailed resource documentation
- [Data Source Schema](./docs/data-sources/diagram.md) - Data source reference

## Examples

The [examples](./examples) directory contains complete Terraform configurations demonstrating:

- **[AWS Infrastructure](./examples/basic-aws/main.tf)** - VPC, EC2, RDS, and more
- **[Azure Resources](./examples/basic-azure/main.tf)** - Virtual networks, VMs, and databases
- **[DigitalOcean Stack](./examples/basic-digitalocean/main.tf)** - Droplets, load balancers, and Spaces
- **[Data Source Usage](./examples/data-sources/main.tf)** - Read-only diagram generation

## Supported Cloud Providers

The provider recognizes and visualizes resources from:

- ✅ **AWS** - EC2, VPC, RDS, S3, ALB, Lambda, and more
- ✅ **Azure** - Virtual Machines, VNets, Storage Accounts, SQL, and more
- ✅ **Google Cloud** - Compute, VPC, Cloud SQL, GCS, and more
- ✅ **DigitalOcean** - Droplets, Load Balancers, Databases, Spaces, and more

## Remote State Backends

The provider automatically detects your Terraform backend configuration and fetches state from remote storage. It supports all major backends with full authentication.

### Supported Backends

 ✅ Fully Supported with Automatic Credential Detection

  | Backend                  | Status         | Authentication Method                     | Credentials Auto-Detection                 |
  |--------------------------|----------------|-------------------------------------------|--------------------------------------------|
  | local                    | ✅ Full Support | File system access                        | N/A (no credentials needed)               |
  | s3 (AWS)                 | ✅ Full Support | AWS SDK v2 with complete credential chain | ✅ Yes - reads from backend  config       |
  | azurerm (Azure)          | ✅ Full Support | Azure SDK with shared key                 | ✅ Yes - reads from backend  config       |
  | remote (Terraform Cloud) | ✅ Full Support | API token authentication                  | ⚠️ Via environment variables  (TFE_TOKEN) |
  | http/https               | ✅ Full Support | Basic authentication                      | ✅ Yes - reads from backend  config       |

  ⚠️ Limited Support

  | Backend            | Status     | Limitation          | Authentication          |
  |--------------------|------------|---------------------|-------------------------|
  | gcs (Google Cloud) | ⚠️ Limited | Public buckets only | HTTP-based (no GCS SDK) |

  ❌ Not Implemented

  | Backend         | Status          | Reason                              |
  |-----------------|-----------------|-------------------------------------|
  | consul          | ❌ Not Supported | No Consul client implementation     |
  | etcdv3          | ❌ Not Supported | No etcd v3 client implementation    |
  | pg (PostgreSQL) | ❌ Not Supported | No PostgreSQL client implementation |

### AWS S3 Backend

The provider automatically reads credentials from your backend configuration:

```hcl
terraform {
  backend "s3" {
    bucket = "my-terraform-state"
    key    = "prod/terraform.tfstate"
    region = "us-east-1"

    # Credentials specified here are automatically used!
    # Option 1: Direct credentials
    # access_key = "AKIAIOSFODNN7EXAMPLE"
    # secret_key = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"

    # Option 2: AWS profile
    # profile = "production"

    # Option 3: Environment variables or IAM role
  }
}

# Simple provider config - no credential duplication!
provider "cartography" {}

data "cartography_diagram" "infra" {
  config_path = path.module  # Auto-detects S3 backend
  output_path = "diagram.svg"
}
```

**Credential Auto-Detection Priority:**
1. Backend config (`access_key`/`secret_key` or `profile` in backend block)
2. Environment variables (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`)
3. Shared credentials file (`~/.aws/credentials`)
4. IAM role (EC2, ECS, Lambda)

### Azure Blob Storage Backend

The provider automatically reads credentials from your backend configuration:

```hcl
terraform {
  backend "azurerm" {
    storage_account_name = "tfstate12345"
    container_name       = "tfstate"
    key                  = "prod.terraform.tfstate"

    # Credentials specified here are automatically used!
    # Option 1: Direct access key
    # access_key = "your-storage-account-key"

    # Option 2: Environment variables
    # ARM_ACCESS_KEY or AZURE_STORAGE_KEY
  }
}

# Simple provider config - no credential duplication!
provider "cartography" {}

data "cartography_diagram" "infra" {
  config_path = path.module  # Auto-detects Azure backend
  output_path = "diagram.svg"
}
```

**Credential Auto-Detection Priority:**
1. Backend config (`access_key` in backend block)
2. Environment variable `ARM_ACCESS_KEY`
3. Environment variable `AZURE_STORAGE_KEY`

### Terraform Cloud/Enterprise Backend

```hcl
terraform {
  backend "remote" {
    organization = "my-org"
    workspaces {
      name = "production"
    }
  }
}

provider "cartography" {
  terraform_token = var.tfe_token  # Or use TFE_TOKEN env var
}

data "cartography_diagram" "infra" {
  config_path = path.module  # Auto-detects TFC backend
  output_path = "diagram.svg"
}
```

### Backend Auto-Detection

When you use `config_path` instead of `state_path`, the provider:
1. **Parses your Terraform configuration files** to find the backend configuration
2. **Extracts credentials** directly from the backend block (access_key, profile, etc.)
3. **Falls back to environment variables** if not specified in backend config
4. **Fetches state** using the detected credentials
5. **Generates your diagram** automatically

**Key Benefit:** No need to duplicate credentials! Just configure your backend once, and the provider reads everything from there.

```hcl
terraform {
  backend "s3" {
    bucket     = "my-state-bucket"
    key        = "terraform.tfstate"
    region     = "us-east-1"
    access_key = "..." # Provider reads this automatically!
    secret_key = "..." # Provider reads this automatically!
  }
}

provider "cartography" {} # That's it!

data "cartography_diagram" "infra" {
  config_path = path.module
  output_path = "diagram.svg"
}
```

See [examples/backend-configurations](./examples/backend-configurations) for complete examples of all supported backends.

## Building from Source

If you want to build the provider from source:

```bash
# Clone the repository
git clone https://github.com/ankek/terraform-provider-cartography
cd terraform-provider-cartography

# Build the provider
go build -o terraform-provider-cartography

# Install locally for testing
mkdir -p ~/.terraform.d/plugins/registry.terraform.io/ankek/cartography/0.1.0/linux_amd64
cp terraform-provider-cartography ~/.terraform.d/plugins/registry.terraform.io/ankek/cartography/0.1.0/linux_amd64/
```

### Development Requirements

- [Go](https://golang.org/doc/install) 1.24+
- [Terraform](https://www.terraform.io/downloads.html) 1.0+
- [terraform-plugin-docs](https://github.com/hashicorp/terraform-plugin-docs) (for documentation generation)

### Running Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/parser
go test ./internal/renderer
```

### Generating Documentation

```bash
# Install tfplugindocs
go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest

# Generate provider documentation
tfplugindocs generate
```

## Contributing

We welcome contributions! Please see our [contributing guidelines](./CONTRIBUTING.md) for:

- How to report bugs
- How to submit pull requests
- Code style and testing requirements
- Development workflow

## License

This project is licensed under the MIT License - see the [LICENSE](./LICENSE) file for details.

## Acknowledgments

- Built with [terraform-plugin-framework](https://github.com/hashicorp/terraform-plugin-framework)
- Diagram rendering powered by SVG and image processing libraries
- Inspired by infrastructure visualization needs across cloud platforms

---

**Note:** This provider is community-maintained and not officially supported by HashiCorp.
