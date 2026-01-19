package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func runTofuSilent(dir string, args ...string) error {
	cmd := exec.Command("tofu", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s", string(output))
	}
	return nil
}

func verifyProvider(name string, tfContent string) {
	fmt.Printf("Testing %s credentials... ", name)

	// Create temp dir
	tmpDir, err := os.MkdirTemp("", "verify_creds_"+name)
	if err != nil {
		fmt.Printf("❌ Error creating temp dir: %v\n", err)
		return
	}
	defer os.RemoveAll(tmpDir)

	// Write main.tf
	if err := os.WriteFile(filepath.Join(tmpDir, "main.tf"), []byte(tfContent), 0644); err != nil {
		fmt.Printf("❌ Error writing main.tf: %v\n", err)
		return
	}

	// Init
	if err := runTofuSilent(tmpDir, "init"); err != nil {
		fmt.Printf("❌ Init failed:\n%v\n", err)
		return
	}

	// Plan
	if err := runTofuSilent(tmpDir, "plan"); err != nil {
		// Clean up error message for display
		msg := err.Error()
		if strings.Contains(msg, "Error:") {
			parts := strings.Split(msg, "Error:")
			if len(parts) > 1 {
				msg = strings.TrimSpace(parts[1])
			}
		}
		fmt.Printf("❌ Plan failed: %s\n", msg)
		return
	}

	fmt.Println("✅ Success!")
}

func runVerifyCreds() {
	// 1. AWS
	verifyProvider("AWS", "\n\t\tprovider \"aws\" {\n\t\t\tregion = \"us-east-1\"\n\t\t}\n\t\tdata \"aws_caller_identity\" \"current\" {}\n\t")

	// 2. Azure
	verifyProvider("Azure", "\n\t\tprovider \"azurerm\" {\n\t\t\tfeatures {} \n\t\t}\n\t\tdata \"azurerm_client_config\" \"current\" {}\n\t")

	// 3. GCP
	verifyProvider("GCP", "\n\t\tprovider \"google\" {\n\t\t\tregion = \"us-central1\"\n\t\t}\n\t\tdata \"google_client_config\" \"current\" {}\n\t")
}
