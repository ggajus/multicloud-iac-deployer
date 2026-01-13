package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/xeipuuv/gojsonschema"
)

func loadSchema(schemaPath string) (*gojsonschema.Schema, error) {
	absPath, err := filepath.Abs(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("error getting absolute path: %w", err)
	}

	schemaLoader := gojsonschema.NewReferenceLoader("file://" + absPath)
	schema, err := gojsonschema.NewSchema(schemaLoader)
	if err != nil {
		return nil, fmt.Errorf("error loading schema: %w", err)
	}

	return schema, nil
}

func validateConfig(schema *gojsonschema.Schema, config map[string]interface{}) (bool, string) {
	configJSON, err := json.Marshal(config)
	if err != nil {
		return false, fmt.Sprintf("error marshaling config: %v", err)
	}

	documentLoader := gojsonschema.NewBytesLoader(configJSON)
	result, err := schema.Validate(documentLoader)
	if err != nil {
		return false, fmt.Sprintf("validation error: %v", err)
	}

	if !result.Valid() {
		var errors []string
		for _, desc := range result.Errors() {
			errors = append(errors, desc.String())
		}
		return false, strings.Join(errors, "; ")
	}

	// GCP-specific validation
	provider, _ := config["provider"].(string)
	if provider == "gcp" {
		services, _ := config["services"].([]interface{})
		for _, svc := range services {
			service, ok := svc.(map[string]interface{})
			if !ok {
				continue
			}
			serviceType, _ := service["type"].(string)
			if serviceType == "compute.instance" {
				if _, hasProjectID := service["project_id"]; !hasProjectID {
					return false, "GCP compute.instance requires 'project_id' in service configuration"
				}
			}
		}
	}

	return true, ""
}

func formatTfvarsValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		return fmt.Sprintf(`"%s"`, v)
	case float64:
		if v == float64(int64(v)) {
			return fmt.Sprintf("%d", int64(v))
		}
		return fmt.Sprintf("%g", v)
	case map[string]interface{}:
		var items []string
		for k, val := range v {
			items = append(items, fmt.Sprintf("  %s = %s", k, formatTfvarsValue(val)))
		}
		return "{\n" + strings.Join(items, "\n") + "\n}"
	case []interface{}:
		var items []string
		for _, item := range v {
			items = append(items, formatTfvarsValue(item))
		}
		return "[" + strings.Join(items, ", ") + "]"
	default:
		return fmt.Sprintf("%v", v)
	}
}

func generateAWSTfvars(config map[string]interface{}, service map[string]interface{}) string {
	var lines []string

	region, _ := config["region"].(string)
	lines = append(lines, fmt.Sprintf(`region = "%s"`, region))

	instanceID, _ := service["instance_id"].(string)
	lines = append(lines, fmt.Sprintf(`instance_id = "%s"`, instanceID))

	size, _ := service["size"].(string)
	lines = append(lines, fmt.Sprintf(`size = "%s"`, size))

	os, _ := service["os"].(string)
	lines = append(lines, fmt.Sprintf(`os = "%s"`, os))

	if diskSizeGB, ok := service["disk_size_gb"]; ok {
		lines = append(lines, fmt.Sprintf("disk_size_gb = %s", formatTfvarsValue(diskSizeGB)))
	}

	if metadata, ok := service["metadata"].(map[string]interface{}); ok && len(metadata) > 0 {
		lines = append(lines, fmt.Sprintf("metadata = %s", formatTfvarsValue(metadata)))
	}

	if sshKey, ok := service["ssh_public_key"].(string); ok {
		lines = append(lines, fmt.Sprintf(`ssh_public_key = "%s"`, sshKey))
	} else {
		lines = append(lines, `ssh_public_key = ""`)
	}

	return strings.Join(lines, "\n") + "\n"
}

func generateGCPTfvars(config map[string]interface{}, service map[string]interface{}) string {
	var lines []string

	if projectID, ok := service["project_id"].(string); ok {
		lines = append(lines, fmt.Sprintf(`project_id = "%s"`, projectID))
	}

	region, _ := config["region"].(string)
	lines = append(lines, fmt.Sprintf(`region = "%s"`, region))

	instanceID, _ := service["instance_id"].(string)
	lines = append(lines, fmt.Sprintf(`instance_id = "%s"`, instanceID))

	size, _ := service["size"].(string)
	lines = append(lines, fmt.Sprintf(`size = "%s"`, size))

	os, _ := service["os"].(string)
	lines = append(lines, fmt.Sprintf(`os = "%s"`, os))

	if diskSizeGB, ok := service["disk_size_gb"]; ok {
		lines = append(lines, fmt.Sprintf("disk_size_gb = %s", formatTfvarsValue(diskSizeGB)))
	}

	if metadata, ok := service["metadata"].(map[string]interface{}); ok && len(metadata) > 0 {
		lines = append(lines, fmt.Sprintf("metadata = %s", formatTfvarsValue(metadata)))
	}

	return strings.Join(lines, "\n") + "\n"
}

func generateAzureTfvars(config map[string]interface{}, service map[string]interface{}) string {
	var lines []string

	region, _ := config["region"].(string)
	lines = append(lines, fmt.Sprintf(`location = "%s"`, region))

	instanceID, _ := service["instance_id"].(string)
	lines = append(lines, fmt.Sprintf(`instance_id = "%s"`, instanceID))

	size, _ := service["size"].(string)
	lines = append(lines, fmt.Sprintf(`size = "%s"`, size))

	os, _ := service["os"].(string)
	lines = append(lines, fmt.Sprintf(`os = "%s"`, os))

	if diskSizeGB, ok := service["disk_size_gb"]; ok {
		lines = append(lines, fmt.Sprintf("disk_size_gb = %s", formatTfvarsValue(diskSizeGB)))
	}

	if metadata, ok := service["metadata"].(map[string]interface{}); ok && len(metadata) > 0 {
		lines = append(lines, fmt.Sprintf("metadata = %s", formatTfvarsValue(metadata)))
	}

	if adminUsername, ok := service["admin_username"].(string); ok {
		lines = append(lines, fmt.Sprintf(`admin_username = "%s"`, adminUsername))
	}

	if sshKey, ok := service["ssh_public_key"].(string); ok {
		lines = append(lines, fmt.Sprintf(`ssh_public_key = "%s"`, sshKey))
	}

	return strings.Join(lines, "\n") + "\n"
}

func getServiceFolderName(serviceType string) string {
	switch serviceType {
	case "compute.instance":
		return "compute_instance"
	case "storage.object":
		return "storage_object"
	}
	return strings.ReplaceAll(serviceType, ".", "_")
}

func parse(configPath string) bool {
	// Load configuration
	configData, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Printf("[ERROR] Error loading configuration: %v\n", err)
		return false
	}

	var config map[string]interface{}
	if err := json.Unmarshal(configData, &config); err != nil {
		fmt.Printf("[ERROR] Error parsing configuration JSON: %v\n", err)
		return false
	}

	// Load and validate schema
	schema, err := loadSchema("schema.json")
	if err != nil {
		fmt.Printf("[ERROR] %v\n", err)
		return false
	}

	isValid, errorMsg := validateConfig(schema, config)
	if !isValid {
		fmt.Printf("[ERROR] Validation failed: %s\n", errorMsg)
		return false
	}

	fmt.Println("✓ Configuration validated successfully")

	// Generate version dir based on timestamp
	versionDir := fmt.Sprintf("tfvars_%s", time.Now().Format("20060102_150405"))

	provider, _ := config["provider"].(string)
	services, _ := config["services"].([]interface{})

	outputDir := filepath.Join("..", "deployment", provider, versionDir)

	// Process each service
	for _, svc := range services {
		service, ok := svc.(map[string]interface{})
		if !ok {
			continue
		}

		if err := os.MkdirAll(outputDir, 0755); err != nil {
			fmt.Printf("[ERROR] Error creating directory %s: %v\n", outputDir, err)
			return false
		}

		// Generate .tfvars content based on provider
		var tfvarsContent string
		switch provider {
		case "aws":
			tfvarsContent = generateAWSTfvars(config, service)
		case "gcp":
			tfvarsContent = generateGCPTfvars(config, service)
		case "azure":
			tfvarsContent = generateAzureTfvars(config, service)
		default:
			fmt.Printf("[ERROR] Unknown provider %s\n", provider)
			return false
		}

		// Name .tfvars file based on service
		tfvarsFile := filepath.Join(outputDir, fmt.Sprintf("%s.tfvars", getServiceFolderName(service["type"].(string))))
		if err := os.WriteFile(tfvarsFile, []byte(tfvarsContent), 0644); err != nil {
			fmt.Printf("[ERROR] Error writing file %s: %v\n", tfvarsFile, err)
			return false
		}

		fmt.Printf("✓ Generated: %s\n", tfvarsFile)
	}

	fmt.Printf("\n✓ Successfully parsed configuration\n")
	fmt.Printf("  Output directory: %s\n", outputDir)
	return true
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run parser.go <config.json>")
		os.Exit(1)
	}

	configPath := os.Args[1]

	if !parse(configPath) {
		os.Exit(1)
	}
}
