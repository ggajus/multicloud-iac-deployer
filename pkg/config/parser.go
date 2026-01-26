package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

type ResourcePlan struct {
	ID        string
	Type      string
	TfVars    string
	ModuleDir string
}

type DeploymentPlan struct {
	Provider  string
	Region    string
	OutputDir string
	Resources []ResourcePlan
}

type Service struct {
	Type          string            `json:"type"`
	InstanceID    string            `json:"instance_id,omitempty"`
	BucketID      string            `json:"bucket_id,omitempty"`
	Size          string            `json:"size,omitempty"`
	OS            string            `json:"os,omitempty"`
	DiskSizeGB    int               `json:"disk_size_gb,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	ProjectID     string            `json:"project_id,omitempty"`
	SSHPublicKey  string            `json:"ssh_public_key,omitempty"`
	AdminUsername string            `json:"admin_username,omitempty"`
	StorageTier   string            `json:"storage_tier,omitempty"`
	Versioning    bool              `json:"versioning,omitempty"`
	// Additional fields can be added here as needed
}

type Config struct {
	Provider       string    `json:"provider"`
	Region         string    `json:"region"`
	ProjectName    string    `json:"project_name"`
	Services       []Service `json:"services"`
	SubscriptionID string    `json:"subscription_id,omitempty"`
	Version        string    `json:"version,omitempty"`
}

var generatorConfig map[string]map[string][]AttributeConfig

func LoadGeneratorConfig(rootPath string) error {
	path := filepath.Join(rootPath, "parser", "generator_config.json")
	configData, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("error loading generator config from %s: %w", path, err)
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

func validateConfig(schema *gojsonschema.Schema, configData []byte) (bool, string) {
	documentLoader := gojsonschema.NewBytesLoader(configData)
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

	// Unmarshal to struct for extra validation logic
	var config Config
	if err := json.Unmarshal(configData, &config); err != nil {
		return false, fmt.Sprintf("error unmarshaling for custom validation: %v", err)
	}

	// provider-specific validation
	if config.Provider == "gcp" {
		for _, service := range config.Services {
			if service.Type == "compute.instance" && service.ProjectID == "" {
				return false, "GCP compute.instance requires 'project_id' in service configuration"
			}
		}
	} else if config.Provider == "azure" {
		// Subscription ID is now optional (env var), so no mandatory check here
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
	case int:
		return fmt.Sprintf("%d", v)
	case bool:
		return fmt.Sprintf("%v", v)
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

// structToMap converts a struct to a map[string]interface{} using JSON marshaling
func structToMap(v interface{}) (map[string]interface{}, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var m map[string]interface{}
	err = json.Unmarshal(data, &m)
	return m, err
}

func generateTfvars(provider, serviceType string, config Config, service Service) (string, error) {
	var lines []string

	// Get attribute configuration based on provider & service
	providerConfig, ok := generatorConfig[provider]
	if !ok {
		return "", nil
	}

	attrs, ok := providerConfig[serviceType]
	if !ok {
		return "", nil
	}

	// Convert structs to maps for dynamic access
	configMap, err := structToMap(config)
	if err != nil {
		return "", fmt.Errorf("failed to convert config to map: %w", err)
	}
	serviceMap, err := structToMap(service)
	if err != nil {
		return "", fmt.Errorf("failed to convert service to map: %w", err)
	}

	for _, attr := range attrs {
		// Determine source map
		sourceMap := serviceMap
		if attr.Source == "config" {
			sourceMap = configMap
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

	return strings.Join(lines, "\n") + "\n", nil
}

func GetServiceFolderName(serviceType string) string {
	switch serviceType {
	case "compute.instance":
		return "compute_instance"
	case "storage.object":
		return "storage_object"
	}
	return strings.ReplaceAll(serviceType, ".", "_")
}

// GeneratePlan parses the config and returns a deployment plan.
func GeneratePlan(configPath string, rootPath string) (*DeploymentPlan, error) {
	// Load configuration
	configData, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("error loading configuration: %w", err)
	}

	// Load and validate schema
	schemaPath := filepath.Join(rootPath, "parser", "schema.json")
	schema, err := loadSchema(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("error loading schema: %w", err)
	}

	isValid, errorMsg := validateConfig(schema, configData)
	if !isValid {
		return nil, fmt.Errorf("validation failed: %s", errorMsg)
	}

	// Unmarshal into Config struct
	var config Config
	if err := json.Unmarshal(configData, &config); err != nil {
		return nil, fmt.Errorf("error parsing configuration JSON: %w", err)
	}
	
	if config.ProjectName == "" {
		return nil, fmt.Errorf("validation error: 'project_name' is required in configuration")
	}

	// Sanitize project name for directory use
	sanitizedProjectName := strings.ReplaceAll(config.ProjectName, " ", "_")
	sanitizedProjectName = strings.ReplaceAll(sanitizedProjectName, "/", "-")

	outputDir := filepath.Join(rootPath, "deployment", config.Provider, sanitizedProjectName)

	plan := &DeploymentPlan{
		Provider:  config.Provider,
		Region:    config.Region,
		OutputDir: outputDir,
		Resources: []ResourcePlan{},
	}

	// Process each service
	for _, service := range config.Services {
		tfvarsContent, err := generateTfvars(config.Provider, service.Type, config, service)
		if err != nil {
			return nil, fmt.Errorf("error generating tfvars for service type %s: %w", service.Type, err)
		}

		// Determine ID
		var id string
		if service.InstanceID != "" {
			id = service.InstanceID
		} else if service.BucketID != "" {
			id = service.BucketID
		} else {
			// Fallback ID
			id = fmt.Sprintf("%s-%d", GetServiceFolderName(service.Type), len(plan.Resources)+1)
		}

		plan.Resources = append(plan.Resources, ResourcePlan{
			ID:        id,
			Type:      service.Type,
			TfVars:    tfvarsContent,
			ModuleDir: GetServiceFolderName(service.Type),
		})
	}

	return plan, nil
}
