package parser

import "strings"

// Resource represents a parsed Terraform resource
type Resource struct {
	Type       string                 // e.g., "azurerm_virtual_machine", "aws_instance", "digitalocean_droplet"
	Name       string                 // resource name
	Provider   string                 // "azure", "aws", "gcp", "digitalocean"
	Attributes map[string]interface{} // resource attributes

	// Computed fields for graph building
	ID           string   // unique identifier
	Dependencies []string // IDs of resources this depends on
}

// ResourceType categorizes resources for graph layout
type ResourceType int

const (
	ResourceTypeUnknown ResourceType = iota
	ResourceTypeNetwork              // VPC, VNet, Subnets
	ResourceTypeSecurity             // Security Groups, NSG, Firewall Rules
	ResourceTypeCompute              // VMs, EC2, Container instances
	ResourceTypeLoadBalancer         // ALB, NLB, Azure LB
	ResourceTypeStorage              // S3, Blob Storage, Disks
	ResourceTypeDatabase             // RDS, Azure SQL, DynamoDB
	ResourceTypeDNS                  // Route53, Azure DNS
	ResourceTypeCertificate          // TLS Certificates, SSL, Key Vault
	ResourceTypeSecret               // Secrets, Keys, Credentials
	ResourceTypeContainer            // Container Registries, Docker
	ResourceTypeCDN                  // CDN, CloudFront
)

// GetResourceType determines the type category of a resource
func GetResourceType(resourceType string) ResourceType {
	// Azure resources
	azureTypeMap := map[string]ResourceType{
		"azurerm_virtual_network":          ResourceTypeNetwork,
		"azurerm_subnet":                   ResourceTypeNetwork,
		"azurerm_network_security_group":   ResourceTypeSecurity,
		"azurerm_network_security_rule":    ResourceTypeSecurity,
		"azurerm_virtual_machine":          ResourceTypeCompute,
		"azurerm_linux_virtual_machine":    ResourceTypeCompute,
		"azurerm_windows_virtual_machine":  ResourceTypeCompute,
		"azurerm_lb":                       ResourceTypeLoadBalancer,
		"azurerm_lb_backend_address_pool":  ResourceTypeLoadBalancer,
		"azurerm_lb_rule":                  ResourceTypeLoadBalancer,
		"azurerm_storage_account":          ResourceTypeStorage,
		"azurerm_managed_disk":             ResourceTypeStorage,
		"azurerm_sql_server":               ResourceTypeDatabase,
		"azurerm_sql_database":             ResourceTypeDatabase,
		"azurerm_dns_zone":                 ResourceTypeDNS,
		"azurerm_key_vault":                ResourceTypeSecret,
		"azurerm_key_vault_certificate":    ResourceTypeCertificate,
		"azurerm_key_vault_key":            ResourceTypeSecret,
		"azurerm_key_vault_secret":         ResourceTypeSecret,
	}

	// AWS resources
	awsTypeMap := map[string]ResourceType{
		"aws_vpc":                           ResourceTypeNetwork,
		"aws_subnet":                        ResourceTypeNetwork,
		"aws_security_group":                ResourceTypeSecurity,
		"aws_security_group_rule":           ResourceTypeSecurity,
		"aws_network_acl":                   ResourceTypeSecurity,
		"aws_instance":                      ResourceTypeCompute,
		"aws_launch_template":               ResourceTypeCompute,
		"aws_lb":                            ResourceTypeLoadBalancer,
		"aws_alb":                           ResourceTypeLoadBalancer,
		"aws_lb_target_group":               ResourceTypeLoadBalancer,
		"aws_lb_listener":                   ResourceTypeLoadBalancer,
		"aws_s3_bucket":                     ResourceTypeStorage,
		"aws_ebs_volume":                    ResourceTypeStorage,
		"aws_db_instance":                   ResourceTypeDatabase,
		"aws_dynamodb_table":                ResourceTypeDatabase,
		"aws_route53_zone":                  ResourceTypeDNS,
		"aws_route53_record":                ResourceTypeDNS,
		"aws_acm_certificate":               ResourceTypeCertificate,
		"aws_acm_certificate_validation":    ResourceTypeCertificate,
		"aws_iam_server_certificate":        ResourceTypeCertificate,
		"aws_secretsmanager_secret":         ResourceTypeSecret,
		"aws_secretsmanager_secret_version": ResourceTypeSecret,
		"aws_kms_key":                       ResourceTypeSecret,
		"aws_kms_alias":                     ResourceTypeSecret,
	}

	// DigitalOcean resources
	digitaloceanTypeMap := map[string]ResourceType{
		"digitalocean_vpc":                  ResourceTypeNetwork,
		"digitalocean_firewall":             ResourceTypeSecurity,
		"digitalocean_droplet":              ResourceTypeCompute,
		"digitalocean_kubernetes_cluster":   ResourceTypeCompute,
		"digitalocean_app":                  ResourceTypeCompute,
		"digitalocean_loadbalancer":         ResourceTypeLoadBalancer,
		"digitalocean_spaces_bucket":        ResourceTypeStorage,
		"digitalocean_volume":               ResourceTypeStorage,
		"digitalocean_database_cluster":     ResourceTypeDatabase,
		"digitalocean_database_db":          ResourceTypeDatabase,
		"digitalocean_database_replica":     ResourceTypeDatabase,
		"digitalocean_domain":               ResourceTypeDNS,
		"digitalocean_record":               ResourceTypeDNS,
		"digitalocean_certificate":          ResourceTypeCertificate,
		"digitalocean_cdn":                  ResourceTypeCDN,
		"digitalocean_container_registry":   ResourceTypeContainer,
	}

	if rt, ok := azureTypeMap[resourceType]; ok {
		return rt
	}
	if rt, ok := awsTypeMap[resourceType]; ok {
		return rt
	}
	if rt, ok := digitaloceanTypeMap[resourceType]; ok {
		return rt
	}

	return ResourceTypeUnknown
}

// IsCloudInfraResource determines if a resource is actual cloud infrastructure
// Filters out local utilities (tls_private_key, local_file, etc.)
func IsCloudInfraResource(resourceType string) bool {
	// List of non-cloud utility resource types to exclude
	excludedTypes := map[string]bool{
		"tls_private_key":                true,
		"tls_cert_request":               true,
		"tls_locally_signed_cert":        true,
		"tls_self_signed_cert":           true,
		"local_file":                     true,
		"local_sensitive_file":           true,
		"null_resource":                  true,
		"random_id":                      true,
		"random_integer":                 true,
		"random_password":                true,
		"random_pet":                     true,
		"random_shuffle":                 true,
		"random_string":                  true,
		"random_uuid":                    true,
		"time_sleep":                     true,
		"time_static":                    true,
		"time_rotating":                  true,
		"time_offset":                    true,
		"terraform_data":                 true,
		"external":                       true,
		"http":                           true,
		"template_file":                  true,
		"template_dir":                   true,
		"template_cloudinit_config":      true,
		"archive_file":                   true,
	}

	return !excludedTypes[resourceType]
}

// ShouldIncludeInDiagram determines if a resource should be included in the diagram
func ShouldIncludeInDiagram(resource Resource) bool {
	// Exclude non-cloud infrastructure resources
	if !IsCloudInfraResource(resource.Type) {
		return false
	}

	// Exclude data sources (they don't create infrastructure)
	// Note: This is handled during parsing, but double-check

	// Exclude resources with "association" or "attachment" in the name
	// These are typically helper resources that create relationships
	// but don't represent actual infrastructure components
	resourceTypeLower := strings.ToLower(resource.Type)
	if strings.Contains(resourceTypeLower, "_association") &&
	   !strings.Contains(resourceTypeLower, "load_balancer") {
		// Exception: load balancer associations should be kept
		// They represent actual infrastructure relationships
		return false
	}

	return true
}
