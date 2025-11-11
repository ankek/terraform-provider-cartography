package renderer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// IconMapping represents an auto-discovered icon mapping
type IconMapping struct {
	ResourceType string
	IconPath     string
	Provider     string
	Category     string
}

// ScanAndMapIcons automatically scans icon directories and creates mappings
func ScanAndMapIcons(iconBaseDir string) (map[string]map[string]string, error) {
	// Result: provider -> (resourceType -> iconPath)
	mappings := make(map[string]map[string]string)
	mappings["azure"] = make(map[string]string)
	mappings["aws"] = make(map[string]string)
	mappings["digitalocean"] = make(map[string]string)
	mappings["gcp"] = make(map[string]string)

	// Scan each provider directory
	providers := []string{"azure", "aws", "digitalocean", "gcp"}
	for _, provider := range providers {
		providerDir := filepath.Join(iconBaseDir, provider)
		if _, err := os.Stat(providerDir); os.IsNotExist(err) {
			continue
		}

		iconFiles, err := findIconFiles(providerDir)
		if err != nil {
			fmt.Printf("Warning: failed to scan %s icons: %v\n", provider, err)
			continue
		}

		// Create mappings for this provider
		for _, iconFile := range iconFiles {
			resourceTypes := guessResourceTypes(provider, iconFile)
			for _, resourceType := range resourceTypes {
				// Get relative path from provider directory
				relPath, err := filepath.Rel(iconBaseDir, iconFile)
				if err != nil {
					continue
				}
				mappings[provider][resourceType] = relPath
			}
		}
	}

	return mappings, nil
}

// findIconFiles recursively finds all icon files in a directory
func findIconFiles(dir string) ([]string, error) {
	var iconFiles []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Check if it's an icon file (SVG or PNG)
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".svg" || ext == ".png" {
			iconFiles = append(iconFiles, path)
		}

		return nil
	})

	return iconFiles, err
}

// guessResourceTypes attempts to map an icon file to Terraform resource types
func guessResourceTypes(provider, iconPath string) []string {
	fileName := filepath.Base(iconPath)
	fileNameLower := strings.ToLower(fileName)

	// Remove extension
	nameWithoutExt := strings.TrimSuffix(fileNameLower, filepath.Ext(fileNameLower))

	// Remove common prefixes/suffixes
	nameWithoutExt = strings.TrimPrefix(nameWithoutExt, "icon-service-")
	nameWithoutExt = strings.TrimPrefix(nameWithoutExt, "icon-")
	nameWithoutExt = strings.TrimSuffix(nameWithoutExt, "-icon")

	// Clean up the name
	cleanName := normalizeIconName(nameWithoutExt)

	var resourceTypes []string

	switch provider {
	case "azure":
		resourceTypes = mapAzureIcon(cleanName, fileNameLower)
	case "aws":
		resourceTypes = mapAWSIcon(cleanName, fileNameLower)
	case "digitalocean":
		resourceTypes = mapDigitalOceanIcon(cleanName, fileNameLower)
	case "gcp":
		resourceTypes = mapGCPIcon(cleanName, fileNameLower)
	}

	return resourceTypes
}

// normalizeIconName converts icon file names to a normalized format
func normalizeIconName(name string) string {
	// Replace hyphens and underscores with spaces
	name = strings.ReplaceAll(name, "-", " ")
	name = strings.ReplaceAll(name, "_", " ")

	// Remove common numbers/codes at start
	parts := strings.Fields(name)
	var cleanParts []string
	for _, part := range parts {
		// Skip numeric prefixes like "03565", "030777508"
		if len(part) > 4 && isNumeric(part[:4]) {
			continue
		}
		cleanParts = append(cleanParts, part)
	}

	return strings.Join(cleanParts, " ")
}

// isNumeric checks if a string contains only digits
func isNumeric(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// mapAzureIcon maps Azure icon files to resource types
func mapAzureIcon(cleanName, fileName string) []string {
	var types []string

	// Common Azure resource mappings
	mappings := map[string][]string{
		"virtual machine":         {"azurerm_virtual_machine", "azurerm_linux_virtual_machine", "azurerm_windows_virtual_machine"},
		"virtual machines":        {"azurerm_virtual_machine", "azurerm_linux_virtual_machine", "azurerm_windows_virtual_machine"},
		"vm":                      {"azurerm_virtual_machine", "azurerm_linux_virtual_machine", "azurerm_windows_virtual_machine"},
		"virtual network":         {"azurerm_virtual_network"},
		"virtual networks":        {"azurerm_virtual_network"},
		"vnet":                    {"azurerm_virtual_network"},
		"subnet":                  {"azurerm_subnet"},
		"subnets":                 {"azurerm_subnet"},
		"network security group":  {"azurerm_network_security_group"},
		"network security groups": {"azurerm_network_security_group"},
		"nsg":                     {"azurerm_network_security_group"},
		"load balancer":           {"azurerm_lb"},
		"load balancers":          {"azurerm_lb"},
		"storage account":         {"azurerm_storage_account"},
		"storage accounts":        {"azurerm_storage_account"},
		"managed disk":            {"azurerm_managed_disk"},
		"managed disks":           {"azurerm_managed_disk"},
		"sql database":            {"azurerm_sql_server", "azurerm_sql_database"},
		"sql databases":           {"azurerm_sql_server", "azurerm_sql_database"},
		"dns zone":                {"azurerm_dns_zone"},
		"dns zones":               {"azurerm_dns_zone"},
		"public ip":               {"azurerm_public_ip"},
		"public ip address":       {"azurerm_public_ip"},
		"public ip addresses":     {"azurerm_public_ip"},
		"network interface":       {"azurerm_network_interface"},
		"network interfaces":      {"azurerm_network_interface"},
		"nic":                     {"azurerm_network_interface"},
		"application gateway":     {"azurerm_application_gateway"},
		"application gateways":    {"azurerm_application_gateway"},
		"vpn gateway":             {"azurerm_vpn_gateway"},
		"vpn gateways":            {"azurerm_vpn_gateway"},
		"firewall":                {"azurerm_firewall"},
		"firewalls":               {"azurerm_firewall"},
		"cosmos db":               {"azurerm_cosmosdb_account"},
		"cosmosdb":                {"azurerm_cosmosdb_account"},
		"postgresql":              {"azurerm_postgresql_server"},
		"mysql":                   {"azurerm_mysql_server"},
		"kubernetes":              {"azurerm_kubernetes_cluster"},
		"aks":                     {"azurerm_kubernetes_cluster"},
		"container instance":      {"azurerm_container_group"},
		"container instances":     {"azurerm_container_group"},
		"app service":             {"azurerm_app_service"},
		"app services":            {"azurerm_app_service"},
		"web app":                 {"azurerm_app_service"},
		"function app":            {"azurerm_function_app"},
		"function apps":           {"azurerm_function_app"},
		"key vault":               {"azurerm_key_vault"},
		"key vaults":              {"azurerm_key_vault"},
	}

	// Check for matches
	for key, resourceTypes := range mappings {
		if strings.Contains(cleanName, key) {
			types = append(types, resourceTypes...)
			break
		}
	}

	return types
}

// mapAWSIcon maps AWS icon files to resource types
func mapAWSIcon(cleanName, fileName string) []string {
	var types []string

	mappings := map[string][]string{
		"vpc":                       {"aws_vpc"},
		"subnet":                    {"aws_subnet"},
		"security group":            {"aws_security_group"},
		"ec2":                       {"aws_instance"},
		"elastic compute cloud":     {"aws_instance"},
		"instance":                  {"aws_instance"},
		"load balancing":            {"aws_lb", "aws_alb"},
		"elastic load balancing":    {"aws_lb", "aws_alb"},
		"alb":                       {"aws_lb", "aws_alb"},
		"nlb":                       {"aws_lb"},
		"s3":                        {"aws_s3_bucket"},
		"simple storage":            {"aws_s3_bucket"},
		"ebs":                       {"aws_ebs_volume"},
		"elastic block":             {"aws_ebs_volume"},
		"rds":                       {"aws_db_instance"},
		"relational database":       {"aws_db_instance"},
		"dynamodb":                  {"aws_dynamodb_table"},
		"route53":                   {"aws_route53_zone", "aws_route53_record"},
		"route 53":                  {"aws_route53_zone", "aws_route53_record"},
		"lambda":                    {"aws_lambda_function"},
		"elastic kubernetes":        {"aws_eks_cluster"},
		"eks":                       {"aws_eks_cluster"},
		"cloudfront":                {"aws_cloudfront_distribution"},
		"iam":                       {"aws_iam_role", "aws_iam_policy"},
		"nat gateway":               {"aws_nat_gateway"},
		"internet gateway":          {"aws_internet_gateway"},
		"network acl":               {"aws_network_acl"},
	}

	for key, resourceTypes := range mappings {
		if strings.Contains(cleanName, key) {
			types = append(types, resourceTypes...)
			break
		}
	}

	return types
}

// mapDigitalOceanIcon maps DigitalOcean icon files to resource types
func mapDigitalOceanIcon(cleanName, fileName string) []string {
	var types []string

	mappings := map[string][]string{
		"droplet":          {"digitalocean_droplet"},
		"vpc":              {"digitalocean_vpc"},
		"firewall":         {"digitalocean_firewall"},
		"load balancer":    {"digitalocean_loadbalancer"},
		"kubernetes":       {"digitalocean_kubernetes_cluster"},
		"database":         {"digitalocean_database_cluster"},
		"spaces":           {"digitalocean_spaces_bucket"},
		"volume":           {"digitalocean_volume"},
		"dns":              {"digitalocean_domain", "digitalocean_record"},
		"domain":           {"digitalocean_domain"},
		"app platform":     {"digitalocean_app"},
		"cdn":              {"digitalocean_cdn"},
		"certificate":      {"digitalocean_certificate"},
	}

	for key, resourceTypes := range mappings {
		if strings.Contains(cleanName, key) {
			types = append(types, resourceTypes...)
			break
		}
	}

	return types
}

// mapGCPIcon maps GCP icon files to resource types
func mapGCPIcon(cleanName, fileName string) []string {
	var types []string

	mappings := map[string][]string{
		"compute engine":     {"google_compute_instance"},
		"vpc":                {"google_compute_network"},
		"subnet":             {"google_compute_subnetwork"},
		"firewall":           {"google_compute_firewall"},
		"load balancing":     {"google_compute_forwarding_rule"},
		"cloud storage":      {"google_storage_bucket"},
		"gcs":                {"google_storage_bucket"},
		"cloud sql":          {"google_sql_database_instance"},
		"kubernetes":         {"google_container_cluster"},
		"gke":                {"google_container_cluster"},
	}

	for key, resourceTypes := range mappings {
		if strings.Contains(cleanName, key) {
			types = append(types, resourceTypes...)
			break
		}
	}

	return types
}

// UpdateIconMaps updates the global icon maps with scanned mappings
func UpdateIconMaps(scannedMappings map[string]map[string]string) {
	if azure, ok := scannedMappings["azure"]; ok {
		for resourceType, iconPath := range azure {
			azureIconMap[resourceType] = iconPath
		}
	}

	if aws, ok := scannedMappings["aws"]; ok {
		for resourceType, iconPath := range aws {
			awsIconMap[resourceType] = iconPath
		}
	}

	if do, ok := scannedMappings["digitalocean"]; ok {
		for resourceType, iconPath := range do {
			digitaloceanIconMap[resourceType] = iconPath
		}
	}

	if gcp, ok := scannedMappings["gcp"]; ok {
		for resourceType, iconPath := range gcp {
			gcpIconMap[resourceType] = iconPath
		}
	}
}

// InitializeIcons scans and initializes icon mappings
func InitializeIcons() error {
	iconBaseDir := "internal/renderer/icons"

	mappings, err := ScanAndMapIcons(iconBaseDir)
	if err != nil {
		return fmt.Errorf("failed to scan icons: %w", err)
	}

	UpdateIconMaps(mappings)

	// Print statistics
	fmt.Printf("Icon auto-mapping complete:\n")
	fmt.Printf("  Azure: %d mappings\n", len(azureIconMap))
	fmt.Printf("  AWS: %d mappings\n", len(awsIconMap))
	fmt.Printf("  DigitalOcean: %d mappings\n", len(digitaloceanIconMap))
	fmt.Printf("  GCP: %d mappings\n", len(gcpIconMap))

	return nil
}
