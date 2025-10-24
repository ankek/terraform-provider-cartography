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

The provider can read state from remote backends by configuring the appropriate credentials:

### AWS S3 Backend
```hcl
provider "cartography" {
  aws_access_key = var.aws_access_key  # Or use AWS_ACCESS_KEY_ID env var
  aws_secret_key = var.aws_secret_key  # Or use AWS_SECRET_ACCESS_KEY env var
}
```

### Azure Blob Storage
```hcl
provider "cartography" {
  azure_account = "mystorageaccount"
  azure_key     = var.azure_key  # Or use ARM_ACCESS_KEY env var
}
```

### Google Cloud Storage
```hcl
provider "cartography" {
  gcp_credentials = file("~/.gcp/credentials.json")  # Or use GOOGLE_APPLICATION_CREDENTIALS env var
}
```

### Terraform Cloud
```hcl
provider "cartography" {
  terraform_token = var.tfe_token  # Or use TFE_TOKEN env var
}
```

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
