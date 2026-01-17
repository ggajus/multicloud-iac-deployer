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

type AttributeConfig struct {
	Field     string      `json:"field"`
	Source    string      `json:"source"`
	Required  bool        `json:"required"`
	Mapping   string      `json:"mapping"`
	SkipEmpty bool        `json:"skip_empty"`
	Default   interface{} `json:"default"`
}

var generatorConfig map[string]map[string][]AttributeConfig

func loadGeneratorConfig() error {
	configData, err := os.ReadFile("generator_config.json")
	if err != nil {
		return fmt.Errorf("error loading generator config: %w", err)
	}

	if err := json.Unmarshal(configData, &generatorConfig); err != nil {
		return fmt.Errorf("error parsing generator config: %w", err)
	}

	return nil
}

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

	provider, _ := config["provider"].(string)

	// provider-specific validation
	if provider == "gcp" { // GCP
		services, _ := config["services"].([]interface{})
		for _, svc := range services {
			service, ok := svc.(map[string]interface{})
			if !ok {
				continue
			}
			if _, hasProjectID := service["project_id"]; !hasProjectID {
				return false, "GCP compute.instance requires 'project_id' in service configuration"
			}
		}
	} else if provider == "azure" { // Azure
		services, _ := config["services"].([]interface{})
		for _, svc := range services {
			service, ok := svc.(map[string]interface{})
			if !ok {
				continue
			}
			if _, hasSubId := service["subscription_id"]; !hasSubId {
				return false, "GCP compute.instance requires 'subscription_id' in service configuration"
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

func generateTfvars(provider, serviceType string, config, service map[string]interface{}) string {
	var lines []string

	// Get attribute configuration based on provider & service
	providerConfig, ok := generatorConfig[provider]
	if !ok {
		return ""
	}

	attrs, ok := providerConfig[serviceType]
	if !ok {
		return ""
	}

	for _, attr := range attrs {
		// Determine source map
		sourceMap := service
		if attr.Source == "config" {
			sourceMap = config
		}

		// Get source field name (use mapping if specified, otherwise use field name)
		sourceField := attr.Field
		if attr.Mapping != "" {
			sourceField = attr.Mapping
		}

		value, exists := sourceMap[sourceField]

		// Handle default values
		if !exists && attr.Default != nil {
			value = attr.Default
			exists = true
		}

		// Skip if not exists
		if !exists {
			continue
		}

		// Skip empty values if specified
		if attr.SkipEmpty {
			if strVal, ok := value.(string); ok && strVal == "" {
				continue
			}
			if mapVal, ok := value.(map[string]interface{}); ok && len(mapVal) == 0 {
				continue
			}
		}

		// Format and append the line
		formattedValue := formatTfvarsValue(value)
		lines = append(lines, fmt.Sprintf("%s = %s", attr.Field, formattedValue))
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
		serviceType, _ := service["type"].(string)
		tfvarsContent := generateTfvars(provider, serviceType, config, service)

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

	// Load generator configuration
	if err := loadGeneratorConfig(); err != nil {
		fmt.Printf("[ERROR] %v\n", err)
		os.Exit(1)
	}

	configPath := os.Args[1]

	if !parse(configPath) {
		os.Exit(1)
	}
}
