# OpenTofu Module & Orchestration Guide

## 1. Module Interface
Each provider module (e.g., `opentofu/gcp/compute_instance/`) is self-contained.

* **`variables.tf`**: Source of truth. Contains input definitions, validation rules, and descriptions.
* **`test.tfvars`**: Example payload. 

## 2. Tofu Commands
Execute these inside the deployment folder:

```bash
# 1. Initialize backend
tofu init

# 2. Plan (saves plan to file)
tofu plan -var-file="run.tfvars" -out=tfplan

# 3. Apply
tofu apply "tfplan"

# 4. Parse Outputs
tofu output -json

```

## 3. Orchestration Strategy

### Folder Isolation

Never run concurrent deployments in the same directory.

1. Create unique directory: `data/jobs/{job_id}/{provider}/`
2. Copy `.tf` templates into this directory.
3. Generate `run.tfvars` locally.
4. Run `tofu` commands inside this isolated path.

### Meta-State Database

The Orchestrator must maintain a database linking the logical request to physical paths.

**Example DB Record:**

```json
{
  "deployment_id": "redundant-web-01",
  "status": "active",
  "sub_deployments": [
    {
      "provider": "gcp",
      "path": "./data/jobs/redundant-web-01/gcp/",
      "state_file": "./data/jobs/redundant-web-01/gcp/terraform.tfstate",
      "status": "success",
      "outputs": { "ip": "34.1.1.1" }
    },
    {
      "provider": "azure",
      "path": "./data/jobs/redundant-web-01/azure/",
      "state_file": "./data/jobs/redundant-web-01/azure/terraform.tfstate",
      "status": "pending",
      "outputs": {}
    }
  ]
}

```
