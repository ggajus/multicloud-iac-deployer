package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/joho/godotenv"

	"multicloud-iac-provisioner/pkg/config"
)

func runCommand(dir string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	fmt.Printf("➜ Running %s %v in %s\n", name, args, dir)
	return cmd.Run()
}

type TofuOutput struct {
	Value interface{} `json:"value"`
}

func getModuleOutputs(modulePath string) ([]string, error) {
	files, err := os.ReadDir(modulePath)
	if err != nil {
		return nil, err
	}

	var outputs []string
	// Regex to find: output "name" {
	re := regexp.MustCompile(`output\s+\"([\w_-]+)\"\s+\{`)

	for _, file := range files {
		if filepath.Ext(file.Name()) != ".tf" {
			continue
		}

		content, err := os.ReadFile(filepath.Join(modulePath, file.Name()))
		if err != nil {
			return nil, err
		}

		matches := re.FindAllStringSubmatch(string(content), -1)
		for _, match := range matches {
			if len(match) > 1 {
				outputs = append(outputs, match[1])
			}
		}
	}
	return outputs, nil
}

func printOutputs(dir string) error {
	cmd := exec.Command("tofu", "output", "-json")
	cmd.Dir = dir
	outputBytes, err := cmd.Output()
	if err != nil {
		return err
	}

	var outputs map[string]TofuOutput
	if err := json.Unmarshal(outputBytes, &outputs); err != nil {
		return fmt.Errorf("error parsing output json: %w", err)
	}

	if len(outputs) == 0 {
		fmt.Println("  (No outputs found)")
		return nil
	}

	fmt.Println("  Outputs:")
	for key, val := range outputs {
		fmt.Printf("    %s: %v\n", key, val.Value)
	}
	return nil
}

func runOutput(provisionDir string) error {
	absProvisionDir, err := filepath.Abs(provisionDir)
	if err != nil {
		return fmt.Errorf("error getting absolute path: %w", err)
	}

	entries, err := os.ReadDir(absProvisionDir)
	if err != nil {
		return fmt.Errorf("error reading provisioning directory: %w", err)
	}

	fmt.Printf("Retrieving outputs from: %s\n", absProvisionDir)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		fmt.Printf("\n----------------------------------------------------------------\n")
		fmt.Printf("Resource: %s\n", entry.Name())
		fmt.Printf("----------------------------------------------------------------\n")

		resourceDir := filepath.Join(absProvisionDir, entry.Name())
		if err := printOutputs(resourceDir); err != nil {
			fmt.Printf("❌ Error retrieving outputs: %v\n", err)
		}
	}
	fmt.Println("\n================================================================")
	return nil
}

func runProvision(configPath string, rootPath string, skipConfirm bool) error {
	// Generate Plan
	plan, err := config.GeneratePlan(configPath, rootPath)
	if err != nil {
		return fmt.Errorf("error generating plan: %w", err)
	}

	fmt.Printf("✓ Plan generated. Output directory: %s\n", plan.OutputDir)
	
	fmt.Println("\nProvisioning Plan:")
	fmt.Printf("  Cloud Provider: %s\n", plan.Provider)
	fmt.Printf("  Region: %s\n", plan.Region)
	fmt.Printf("  Output Directory: %s\n", plan.OutputDir)
	fmt.Printf("  Resources (%d):\n", len(plan.Resources))
	for _, res := range plan.Resources {
		fmt.Printf("    - %s (Type: %s)\n", res.ID, res.Type)
	}
	fmt.Println()

	if !skipConfirm {
		fmt.Print("Do you want to proceed? [y/N]: ")
		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(response)

		if strings.ToLower(response) != "y" {
			fmt.Println("Provisioning cancelled.")
			return nil
		}
	}

	// Create Output Directory
	if err := os.MkdirAll(plan.OutputDir, 0755); err != nil {
		return fmt.Errorf("error creating output directory: %w", err)
	}

	// Execute Plan
	for _, res := range plan.Resources {
		fmt.Printf("\n----------------------------------------------------------------\n")
		fmt.Printf("Provisioning Resource: %s (Type: %s)\n", res.ID, res.Type)
		fmt.Printf("----------------------------------------------------------------\n")

		targetDir := filepath.Join(plan.OutputDir, res.ID)
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			return fmt.Errorf("error creating resource directory %s: %w", targetDir, err)
		}

		// 1. Resolve Module Path
		moduleSource := filepath.Join(rootPath, "opentofu", plan.Provider, res.ModuleDir)

		// Check if module exists
		if _, err := os.Stat(moduleSource); os.IsNotExist(err) {
			return fmt.Errorf("module not found at %s", moduleSource)
		}

		// Ensure absolute path for the source
		absModuleSource, err := filepath.Abs(moduleSource)
		if err != nil {
			return fmt.Errorf("error getting absolute path for module source: %w", err)
		}

		// Detect outputs from the module
		moduleOutputs, err := getModuleOutputs(absModuleSource)
		if err != nil {
			fmt.Printf("⚠️  Warning: Could not scan module outputs: %v\n", err)
		}

		// Build output forwarding blocks
		var outputBlocks string
		for _, out := range moduleOutputs {
			outputBlocks += fmt.Sprintf(`
output "%s" {
  value = module.provision.%s
}
`, out, out)
		}

		// 2. Generate main.tf with Module Reference AND Output Forwarding
		mainTfContent := fmt.Sprintf(`
module "provision" {
  source = "%s"

%s
}

%s
`, absModuleSource, res.TfVars, outputBlocks)

		mainTfPath := filepath.Join(targetDir, "main.tf")
		if err := os.WriteFile(mainTfPath, []byte(mainTfContent), 0644); err != nil {
			return fmt.Errorf("error writing main.tf for %s: %w", res.ID, err)
		}
		fmt.Printf("✓ Generated main.tf referencing module at %s\n", absModuleSource)

		// 3. Tofu Init
		// We use -upgrade to ensure that if the source path content changed or we are switching dev modes, it updates.
		if err := runCommand(targetDir, "tofu", "init", "-upgrade"); err != nil {
			return fmt.Errorf("error initializing OpenTofu for %s: %w", res.ID, err)
		}

		// 4. Tofu Apply
		if err := runCommand(targetDir, "tofu", "apply", "-auto-approve"); err != nil {
			return fmt.Errorf("error applying OpenTofu for %s: %w", res.ID, err)
		}

		fmt.Printf("✓ Successfully provisioned %s\n", res.ID)

		// 5. Display Outputs
		if err := printOutputs(targetDir); err != nil {
			fmt.Printf("⚠️  Warning: Could not retrieve outputs for %s: %v\n", res.ID, err)
		}
	}

	fmt.Printf("\n================================================================\n")
	fmt.Printf("Provisioning Complete!\n")
	fmt.Printf("State stored in: %s\n", plan.OutputDir)
	return nil
}

func runDestroy(provisionDir string) error {
	absProvisionDir, err := filepath.Abs(provisionDir)
	if err != nil {
		return fmt.Errorf("error getting absolute path: %w", err)
	}

	fmt.Printf("Destroying provisioning at: %s\n", absProvisionDir)

	entries, err := os.ReadDir(absProvisionDir)
	if err != nil {
		return fmt.Errorf("error reading provisioning directory: %w", err)
	}

	allDestroyed := true

	// Iterate over subdirectories (resources)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		resourceDir := filepath.Join(absProvisionDir, entry.Name())
		fmt.Printf("\n----------------------------------------------------------------\n")
		fmt.Printf("Destroying Resource: %s\n", entry.Name())
		fmt.Printf("----------------------------------------------------------------\n")

		// Check if it's a valid tofu directory (has .terraform or .tf files)
		// We'll just assume if it has terraform.tfstate or main.tf it's valid
		// The safest check is trying to run tofu destroy.

		if err := runCommand(resourceDir, "tofu", "destroy", "-auto-approve"); err != nil {
			fmt.Printf("❌ Error destroying %s: %v\n", entry.Name(), err)
			allDestroyed = false
			// Continue destroying other resources even if one fails
			continue
		}

		fmt.Printf("✓ Successfully destroyed %s\n", entry.Name())
	}

	fmt.Printf("\n================================================================\n")
	if allDestroyed {
		fmt.Printf("Destruction Complete! Removing provisioning directory...\n")
		if err := os.RemoveAll(absProvisionDir); err != nil {
			fmt.Printf("❌ Error removing directory: %v\n", err)
		} else {
			fmt.Printf("✓ Removed %s\n", absProvisionDir)
		}
	} else {
		fmt.Printf("⚠️  Destruction finished with errors. Provisioning directory preserved at: %s\n", absProvisionDir)
	}
	
	if !allDestroyed {
		return fmt.Errorf("some resources failed to destroy")
	}
	return nil
}

func main() {
	_ = godotenv.Load()

	if len(os.Args) < 2 {
		fmt.Println("Usage:")
		fmt.Println("  provisioner provision <config.json>")
		fmt.Println("  provisioner output <provisioning_directory>")
		fmt.Println("  provisioner destroy <provisioning_directory>")
		fmt.Println("  provisioner verify-creds")
		os.Exit(1)
	}

	command := os.Args[1]

	// Assuming running from project root
	rootPath, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting working directory: %v\n", err)
		os.Exit(1)
	}

	// Load generator configuration
	if err := config.LoadGeneratorConfig(rootPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error loading generator config: %v\n", err)
		os.Exit(1)
	}

	// Verify critical directories exist
	if _, err := os.Stat(filepath.Join(rootPath, "opentofu")); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "❌ Error: 'opentofu' directory not found in %s.\n", rootPath)
		fmt.Fprintf(os.Stderr, "Please run this tool from the project root directory.\n")
		os.Exit(1)
	}

	switch command {
	case "provision":
		// Parse flags for the provision command
		provisionCmd := flag.NewFlagSet("provision", flag.ExitOnError)
		skipConfirm := provisionCmd.Bool("s", false, "Skip confirmation")

		// Parse arguments excluding the program name and "provision" command
		if err := provisionCmd.Parse(os.Args[2:]); err != nil {
			fmt.Println("Usage: provisioner provision [-s] <config.json>")
			os.Exit(1)
		}

		args := provisionCmd.Args()
		if len(args) < 1 {
			fmt.Println("Usage: provisioner provision [-s] <config.json>")
			os.Exit(1)
		}

		configPath := args[0]

		if err := runProvision(configPath, rootPath, *skipConfirm); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Provisioning failed: %v\n", err)
			os.Exit(1)
		}
	case "output":
		if len(os.Args) < 3 {
			fmt.Println("Usage: provisioner output <provisioning_directory>")
			os.Exit(1)
		}
		if err := runOutput(os.Args[2]); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Output retrieval failed: %v\n", err)
			os.Exit(1)
		}
	case "destroy":
		if len(os.Args) < 3 {
			fmt.Println("Usage: provisioner destroy <provisioning_directory>")
			os.Exit(1)
		}
		if err := runDestroy(os.Args[2]); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Destruction failed: %v\n", err)
			os.Exit(1)
		}
	case "verify-creds":
		runVerifyCreds()
	default:
		// Fallback for backward compatibility or direct config execution
		// If first arg is a file that ends in .json, assume provision
		if filepath.Ext(command) == ".json" {
			if err := runProvision(command, rootPath, false); err != nil {
				fmt.Fprintf(os.Stderr, "❌ Provisioning failed: %v\n", err)
				os.Exit(1)
			}
		} else {
			fmt.Printf("Unknown command: %s\n", command)
			fmt.Println("Usage:")
			fmt.Println("  provisioner provision [-s] <config.json>")
			fmt.Println("  provisioner output <provisioning_directory>")
			fmt.Println("  provisioner destroy <provisioning_directory>")
			fmt.Println("  provisioner verify-creds")
			os.Exit(1)
		}
	}
}
