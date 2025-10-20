terraform {
  required_providers {
    cartography = {
      source  = "ankek/cartography"
      version = "~> 0.1"
    }
  }
}

provider "cartography" {}

# Generate diagram from existing state file
data "cartography_diagram" "from_state" {
  state_path     = "/path/to/terraform.tfstate"
  output_path    = "./diagrams/infrastructure-state.png"
  format         = "png"
  direction      = "TB"
  include_labels = true
  title          = "Infrastructure from State"
}

# Generate diagram from HCL configuration files
data "cartography_diagram" "from_config" {
  config_path    = "/path/to/terraform/configs"
  output_path    = "./diagrams/infrastructure-config.svg"
  format         = "svg"
  direction      = "LR"
  include_labels = false
  title          = "Infrastructure from Config"
}

output "state_diagram_resources" {
  description = "Number of resources in state diagram"
  value       = data.cartography_diagram.from_state.resource_count
}

output "config_diagram_resources" {
  description = "Number of resources in config diagram"
  value       = data.cartography_diagram.from_config.resource_count
}
