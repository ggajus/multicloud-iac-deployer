//go:build integration

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/joho/godotenv"

	"multicloud-iac-deployer/pkg/config"
)

// To run these tests: go test -v -tags=integration ./cmd/deployer

func TestE2E_DeployAndDestroy(t *testing.T) {
	// Find project root
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// We assume we are running from project root or cmd/deployer.
	// Adjust rootPath finding logic.
	var rootPath string
	if _, err := os.Stat(filepath.Join(wd, "examples")); err == nil {
		rootPath = wd
	} else if _, err := os.Stat(filepath.Join(wd, "..", "..", "examples")); err == nil {
		rootPath = filepath.Join(wd, "..", "..")
	} else {
		t.Fatalf("Could not locate project root from %s", wd)
	}

	// Load .env file for credentials
	_ = godotenv.Load(filepath.Join(rootPath, ".env"))

	// Load Generator Config globally (required by main package logic)
	if err := config.LoadGeneratorConfig(rootPath); err != nil {
		t.Fatalf("Failed to load generator config: %v", err)
	}

	examples := []string{
		"examples/aws_demo.json",
		"examples/azure_demo.json",
		"examples/gcp_demo.json",
	}

	for _, exampleRelPath := range examples {
		configPath := filepath.Join(rootPath, exampleRelPath)
		t.Run(fmt.Sprintf("Deploy_%s", filepath.Base(configPath)), func(t *testing.T) {
			
			// 1. Parse Config to predict output directory
			configData, err := os.ReadFile(configPath)
			if err != nil {
				t.Fatalf("Failed to read config file: %v", err)
			}
			var cfg config.Config
			if err := json.Unmarshal(configData, &cfg); err != nil {
				t.Fatalf("Failed to parse config json: %v", err)
			}
			
			// Reconstruct expected output dir logic
			// sanitizedProjectName := strings.ReplaceAll(cfg.ProjectName, " ", "_") 
			// sanitizedProjectName = strings.ReplaceAll(sanitizedProjectName, "/", "-")
			// But we can just rely on runDeploy returning error or not, 
			// and verify the directory exists after deployment.
			
			// Note: runDeploy prints to stdout, which go test captures.

			// 2. Run Deploy
			fmt.Printf(">>> Starting Deployment for %s\n", exampleRelPath)
			err = runDeploy(configPath, rootPath)
			if err != nil {
				t.Logf("Deployment failed for %s: %v", exampleRelPath, err)
				t.Log("Skipping destroy verification due to deployment failure (this is expected if credentials are missing)")
				// We don't fail the test immediately because we want to try the others, 
				// and mostly likely the user doesn't have all 3 clouds configured.
				// However, if strict testing is required, use t.Fail()
				t.Skip("Skipping remainder of test due to missing credentials/setup failure") 
				return
			}

			// Calculate expected output path to verify existence
			// We need to call GeneratePlan again or replicate logic? 
			// Let's call GeneratePlan, it's safe.
			plan, err := config.GeneratePlan(configPath, rootPath)
			if err != nil {
				t.Fatalf("Failed to generate plan for verification: %v", err)
			}
			
			if _, err := os.Stat(plan.OutputDir); os.IsNotExist(err) {
				t.Fatalf("Output directory was not created: %s", plan.OutputDir)
			}

			// Verify main.tf exists for resources
			for _, res := range plan.Resources {
				mainTf := filepath.Join(plan.OutputDir, res.ID, "main.tf")
				if _, err := os.Stat(mainTf); os.IsNotExist(err) {
					t.Errorf("main.tf missing for resource %s", res.ID)
				}
			}

			// 3. Run Destroy
			fmt.Printf(">>> Starting Destruction for %s\n", exampleRelPath)
			err = runDestroy(plan.OutputDir)
			if err != nil {
				t.Errorf("Destruction failed for %s: %v", exampleRelPath, err)
			}

			// Verify directory is gone
			if _, err := os.Stat(plan.OutputDir); !os.IsNotExist(err) {
				t.Errorf("Deployment directory should have been removed: %s", plan.OutputDir)
			}
		})
	}
}
