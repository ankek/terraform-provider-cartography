package renderer

import (
	"embed"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed icons
var embeddedIcons embed.FS

// IconMode determines how icons are loaded
type IconMode int

const (
	IconModeEmbedded IconMode = iota // Use embedded icons from binary
	IconModeExternal                 // Load icons from filesystem
	IconModeDisabled                 // Disable icons, use text only
)

var currentIconMode = IconModeEmbedded

// SetIconMode changes the icon loading mode
func SetIconMode(mode IconMode) {
	currentIconMode = mode
}

// Azure icon mappings (using actual downloaded files)
var azureIconMap = map[string]string{
	"azurerm_virtual_network":                "icons/azure/networking/10061-icon-service-Virtual-Networks.svg",
	"azurerm_subnet":                         "icons/azure/networking/10061-icon-service-Virtual-Networks.svg",
	"azurerm_network_security_group":         "icons/azure/networking/10067-icon-service-Network-Security-Groups.svg",
	"azurerm_network_security_rule":          "icons/azure/networking/10067-icon-service-Network-Security-Groups.svg",
	"azurerm_virtual_machine":                "icons/azure/compute/10021-icon-service-Virtual-Machine.svg",
	"azurerm_linux_virtual_machine":          "icons/azure/compute/10021-icon-service-Virtual-Machine.svg",
	"azurerm_windows_virtual_machine":        "icons/azure/compute/10021-icon-service-Virtual-Machine.svg",
	"azurerm_lb":                             "icons/azure/networking/10062-icon-service-Load-Balancers.svg",
	"azurerm_lb_backend_address_pool":        "icons/azure/networking/10062-icon-service-Load-Balancers.svg",
	"azurerm_lb_rule":                        "icons/azure/networking/10062-icon-service-Load-Balancers.svg",
	"azurerm_storage_account":                "icons/azure/storage/10086-icon-service-Storage-Accounts.svg",
	"azurerm_managed_disk":                   "icons/azure/compute/10032-icon-service-Disks.svg",
	"azurerm_sql_server":                     "icons/azure/databases/02390-icon-service-Azure-SQL.svg",
	"azurerm_sql_database":                   "icons/azure/databases/10130-icon-service-SQL-Database.svg",
	"azurerm_dns_zone":                       "icons/azure/networking/10064-icon-service-DNS-Zones.svg",
	"azurerm_public_ip":                      "icons/azure/networking/10069-icon-service-Public-IP-Addresses.svg",
	"azurerm_network_interface":              "icons/azure/networking/10080-icon-service-Network-Interfaces.svg",
	// Security & Certificates
	"azurerm_key_vault":                      "icons/generic/security.svg",
	"azurerm_key_vault_certificate":          "icons/generic/tls-certificate.svg",
	"azurerm_key_vault_key":                  "icons/generic/private-key.svg",
	"azurerm_key_vault_secret":               "icons/generic/private-key.svg",
}

// AWS icon mappings (using actual downloaded files)
var awsIconMap = map[string]string{
	"aws_vpc":                 "icons/aws/Architecture-Service-Icons_07312025/Arch_Networking-Content-Delivery/64/Arch_Amazon-Virtual-Private-Cloud_64.svg",
	"aws_subnet":              "icons/aws/Architecture-Service-Icons_07312025/Arch_Networking-Content-Delivery/64/Arch_Amazon-Virtual-Private-Cloud_64.svg",
	"aws_security_group":      "icons/aws/Architecture-Service-Icons_07312025/Arch_Security-Identity-Compliance/64/Arch_AWS-Security-Hub_64.svg",
	"aws_security_group_rule": "icons/aws/Architecture-Service-Icons_07312025/Arch_Security-Identity-Compliance/64/Arch_AWS-Security-Hub_64.svg",
	"aws_network_acl":         "icons/aws/Architecture-Service-Icons_07312025/Arch_Security-Identity-Compliance/64/Arch_AWS-Security-Hub_64.svg",
	"aws_instance":            "icons/aws/Architecture-Service-Icons_07312025/Arch_Compute/64/Arch_Amazon-EC2_64.svg",
	"aws_launch_template":     "icons/aws/Architecture-Service-Icons_07312025/Arch_Compute/64/Arch_Amazon-EC2_64.svg",
	"aws_lb":                  "icons/aws/Architecture-Service-Icons_07312025/Arch_Networking-Content-Delivery/64/Arch_Elastic-Load-Balancing_64.svg",
	"aws_alb":                 "icons/aws/Architecture-Service-Icons_07312025/Arch_Networking-Content-Delivery/64/Arch_Elastic-Load-Balancing_64.svg",
	"aws_lb_target_group":     "icons/aws/Architecture-Service-Icons_07312025/Arch_Networking-Content-Delivery/64/Arch_Elastic-Load-Balancing_64.svg",
	"aws_lb_listener":         "icons/aws/Architecture-Service-Icons_07312025/Arch_Networking-Content-Delivery/64/Arch_Elastic-Load-Balancing_64.svg",
	"aws_s3_bucket":           "icons/aws/Architecture-Service-Icons_07312025/Arch_Storage/64/Arch_Amazon-Simple-Storage-Service_64.svg",
	"aws_ebs_volume":          "icons/aws/Architecture-Service-Icons_07312025/Arch_Storage/64/Arch_Amazon-Elastic-Block-Store_64.svg",
	"aws_db_instance":         "icons/aws/Architecture-Service-Icons_07312025/Arch_Database/64/Arch_Amazon-RDS_64.svg",
	"aws_dynamodb_table":      "icons/aws/Architecture-Service-Icons_07312025/Arch_Database/64/Arch_Amazon-DynamoDB_64.svg",
	"aws_route53_zone":        "icons/aws/Architecture-Service-Icons_07312025/Arch_Networking-Content-Delivery/64/Arch_Amazon-Route-53_64.svg",
	"aws_route53_record":      "icons/aws/Architecture-Service-Icons_07312025/Arch_Networking-Content-Delivery/64/Arch_Amazon-Route-53_64.svg",
	// Security & Certificates
	"aws_acm_certificate":               "icons/generic/tls-certificate.svg",
	"aws_acm_certificate_validation":    "icons/generic/certificate-authority.svg",
	"aws_secretsmanager_secret":         "icons/generic/private-key.svg",
	"aws_secretsmanager_secret_version": "icons/generic/private-key.svg",
	"aws_kms_key":                       "icons/generic/private-key.svg",
	"aws_kms_alias":                     "icons/generic/private-key.svg",
	"aws_iam_server_certificate":        "icons/generic/tls-certificate.svg",
}

// DigitalOcean icon mappings
var digitaloceanIconMap = map[string]string{
	"digitalocean_vpc":                  "icons/digitalocean/vpc.svg",
	"digitalocean_firewall":             "icons/digitalocean/firewall.svg",
	"digitalocean_droplet":              "icons/digitalocean/droplet.svg",
	"digitalocean_kubernetes_cluster":   "icons/digitalocean/kubernetes.svg",
	"digitalocean_app":                  "icons/digitalocean/app-platform.svg",
	"digitalocean_loadbalancer":         "icons/digitalocean/load-balancer.svg",
	"digitalocean_spaces_bucket":        "icons/digitalocean/spaces.svg",
	"digitalocean_volume":               "icons/digitalocean/volumes.svg",
	"digitalocean_volume_attachment":    "icons/digitalocean/volumes.svg",
	"digitalocean_database_cluster":     "icons/digitalocean/database.svg",
	"digitalocean_database_db":          "icons/digitalocean/database.svg",
	"digitalocean_database_replica":     "icons/digitalocean/database.svg",
	"digitalocean_domain":               "icons/digitalocean/dns.svg",
	"digitalocean_record":               "icons/digitalocean/dns.svg",
	"digitalocean_cdn":                  "icons/digitalocean/cdn.svg",
	"digitalocean_certificate":          "icons/generic/tls-certificate.svg",
	"digitalocean_container_registry":   "icons/generic/container.svg",
	"digitalocean_ssh_key":              "icons/digitalocean/droplet.svg",
	"digitalocean_monitor_alert":        "icons/generic/monitoring.svg",
}

// GCP icon mappings (placeholder)
var gcpIconMap = map[string]string{
	"google_compute_network":        "icons/gcp/vpc.svg",
	"google_compute_subnetwork":     "icons/gcp/vpc.svg",
	"google_compute_firewall":       "icons/gcp/firewall.svg",
	"google_compute_instance":       "icons/gcp/compute-engine.svg",
	"google_compute_forwarding_rule": "icons/gcp/load-balancing.svg",
	"google_storage_bucket":         "icons/gcp/cloud-storage.svg",
}

// getIconPath returns the path to the icon for a given provider and resource type
func getIconPath(provider, resourceType string) string {
	var iconMap map[string]string

	switch provider {
	case "azure":
		iconMap = azureIconMap
	case "aws":
		iconMap = awsIconMap
	case "digitalocean":
		iconMap = digitaloceanIconMap
	case "gcp":
		iconMap = gcpIconMap
	default:
		return ""
	}

	iconFile, ok := iconMap[resourceType]
	if !ok {
		return ""
	}

	// Icon path already includes icons/provider/ prefix in the map
	return iconFile
}

// getIconData returns the icon data, either from embedded FS or external file
func getIconData(iconPath string) ([]byte, error) {
	if currentIconMode == IconModeDisabled || iconPath == "" {
		return nil, fmt.Errorf("icons disabled or path empty")
	}

	if currentIconMode == IconModeEmbedded {
		// Try to read from embedded filesystem
		data, err := embeddedIcons.ReadFile(iconPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read embedded icon %s: %w", iconPath, err)
		}
		return data, nil
	}

	// IconModeExternal: Read from filesystem
	fullPath := filepath.Join("internal/renderer", iconPath)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read icon file %s: %w", fullPath, err)
	}
	return data, nil
}

// getIconBase64 returns the base64-encoded icon data
func getIconBase64(iconPath string) (string, error) {
	data, err := getIconData(iconPath)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

// getIconDataURI returns a data URI for embedding in SVG/HTML
func getIconDataURI(iconPath string) (string, error) {
	data, err := getIconData(iconPath)
	if err != nil {
		return "", err
	}

	// Determine MIME type based on extension
	mimeType := "image/svg+xml"
	if filepath.Ext(iconPath) == ".png" {
		mimeType = "image/png"
	}

	encoded := base64.StdEncoding.EncodeToString(data)
	return fmt.Sprintf("data:%s;base64,%s", mimeType, encoded), nil
}

// IconExists checks if an icon exists for a given provider and resource type
func IconExists(provider, resourceType string) bool {
	iconPath := getIconPath(provider, resourceType)
	if iconPath == "" {
		return false
	}

	if currentIconMode == IconModeEmbedded {
		_, err := embeddedIcons.ReadFile(iconPath)
		return err == nil
	}

	fullPath := filepath.Join("internal/renderer", iconPath)
	_, err := os.Stat(fullPath)
	return err == nil
}

// GetIconForResource returns the icon path and whether it exists
func GetIconForResource(provider, resourceType string) (string, bool) {
	iconPath := getIconPath(provider, resourceType)
	if iconPath == "" {
		return "", false
	}

	exists := IconExists(provider, resourceType)
	return iconPath, exists
}
