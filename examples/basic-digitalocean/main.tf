terraform {
  required_providers {
    digitalocean = {
      source  = "digitalocean/digitalocean"
      version = "~> 2.0"
    }
    cartography = {
      source  = "antonputra/cartography"
      version = "~> 0.1"
    }
  }
}

provider "digitalocean" {
  # Set DO_TOKEN environment variable
}

provider "cartography" {}

# Sample DigitalOcean infrastructure
resource "digitalocean_vpc" "example" {
  name     = "example-vpc"
  region   = "nyc3"
  ip_range = "10.10.0.0/16"
}

resource "digitalocean_firewall" "web" {
  name = "web-firewall"

  droplet_ids = [digitalocean_droplet.web.id]

  inbound_rule {
    protocol         = "tcp"
    port_range       = "443"
    source_addresses = ["0.0.0.0/0", "::/0"]
  }

  inbound_rule {
    protocol         = "tcp"
    port_range       = "80"
    source_addresses = ["0.0.0.0/0", "::/0"]
  }

  inbound_rule {
    protocol         = "tcp"
    port_range       = "22"
    source_addresses = ["0.0.0.0/0", "::/0"]
  }

  outbound_rule {
    protocol              = "tcp"
    port_range            = "1-65535"
    destination_addresses = ["0.0.0.0/0", "::/0"]
  }

  outbound_rule {
    protocol              = "udp"
    port_range            = "1-65535"
    destination_addresses = ["0.0.0.0/0", "::/0"]
  }
}

resource "digitalocean_loadbalancer" "public" {
  name   = "public-loadbalancer"
  region = "nyc3"
  vpc_uuid = digitalocean_vpc.example.id

  forwarding_rule {
    entry_port      = 443
    entry_protocol  = "https"
    target_port     = 8080
    target_protocol = "http"
    certificate_name = "example-cert"
  }

  forwarding_rule {
    entry_port      = 80
    entry_protocol  = "http"
    target_port     = 8080
    target_protocol = "http"
  }

  healthcheck {
    port     = 8080
    protocol = "http"
    path     = "/health"
  }

  droplet_ids = [
    digitalocean_droplet.web.id,
    digitalocean_droplet.web2.id,
  ]
}

resource "digitalocean_droplet" "web" {
  image    = "ubuntu-22-04-x64"
  name     = "web-1"
  region   = "nyc3"
  size     = "s-1vcpu-1gb"
  vpc_uuid = digitalocean_vpc.example.id

  tags = ["web", "production"]

  ssh_keys = [
    data.digitalocean_ssh_key.terraform.id
  ]

  user_data = <<-EOF
    #!/bin/bash
    apt-get update
    apt-get install -y nginx
    systemctl enable nginx
    systemctl start nginx
  EOF
}

resource "digitalocean_droplet" "web2" {
  image    = "ubuntu-22-04-x64"
  name     = "web-2"
  region   = "nyc3"
  size     = "s-1vcpu-1gb"
  vpc_uuid = digitalocean_vpc.example.id

  tags = ["web", "production"]

  ssh_keys = [
    data.digitalocean_ssh_key.terraform.id
  ]

  user_data = <<-EOF
    #!/bin/bash
    apt-get update
    apt-get install -y nginx
    systemctl enable nginx
    systemctl start nginx
  EOF
}

resource "digitalocean_volume" "data" {
  region                  = "nyc3"
  name                    = "data-volume"
  size                    = 100
  initial_filesystem_type = "ext4"
  description             = "Application data volume"
}

resource "digitalocean_volume_attachment" "data" {
  droplet_id = digitalocean_droplet.web.id
  volume_id  = digitalocean_volume.data.id
}

resource "digitalocean_database_cluster" "postgres" {
  name       = "example-postgres"
  engine     = "pg"
  version    = "15"
  size       = "db-s-1vcpu-1gb"
  region     = "nyc3"
  node_count = 1

  private_network_uuid = digitalocean_vpc.example.id
}

resource "digitalocean_database_db" "app_db" {
  cluster_id = digitalocean_database_cluster.postgres.id
  name       = "app_database"
}

resource "digitalocean_spaces_bucket" "assets" {
  name   = "example-assets"
  region = "nyc3"
  acl    = "private"

  versioning {
    enabled = true
  }
}

resource "digitalocean_domain" "example" {
  name = "example.com"
}

resource "digitalocean_record" "www" {
  domain = digitalocean_domain.example.name
  type   = "A"
  name   = "www"
  value  = digitalocean_loadbalancer.public.ip
  ttl    = 300
}

# Data source for SSH key (you need to create this in DO first)
data "digitalocean_ssh_key" "terraform" {
  name = "terraform"
}

# Generate infrastructure diagram
resource "cartography_diagram" "infrastructure" {
  state_path     = "${path.module}/terraform.tfstate"
  output_path    = "${path.module}/digitalocean-infrastructure.png"
  format         = "png"
  direction      = "TB"
  include_labels = true
  title          = "DigitalOcean Infrastructure Diagram"
}

# Alternative: Generate SVG diagram using data source
data "cartography_diagram" "infrastructure_svg" {
  state_path     = "${path.module}/terraform.tfstate"
  output_path    = "${path.module}/digitalocean-infrastructure.svg"
  format         = "svg"
  direction      = "LR"
  include_labels = true
  title          = "DigitalOcean Infrastructure (Left-to-Right)"
}

output "loadbalancer_ip" {
  value = digitalocean_loadbalancer.public.ip
}

output "diagram_resource_count" {
  value = data.cartography_diagram.infrastructure_svg.resource_count
}
