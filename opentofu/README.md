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

## 4. Supported Primitives

### `compute_instance`

**Common Inputs:**
* `instance_id` (string): Unique identifier.
* `size` (string): `small`, `medium`, `large`.
* `os` (string): `ubuntu`, `debian`.
* `disk_size_gb` (number): Size of the root disk.
* `metadata` (map): Arbitrary tags/labels.

**Provider Specifics:**
* **AWS**: Requires `region`. Optional `ssh_public_key`.
* **Azure**: Requires `subscription_id`, `location`. Optional `admin_username`, `ssh_public_key`.
* **GCP**: Requires `project_id`, `region`.

### `storage_object`

**Common Inputs:**
* `bucket_id` (string): Globally unique name. (Note: Automatically sanitized to lowercase across all providers for compatibility).
* `storage_tier` (string): `standard`, `infrequent`, `cold`, `archive`.
* `versioning` (bool): Enable object versioning.
* `metadata` (map): Arbitrary tags/labels.

**Common Outputs:**
* `bucket_name`: The final, logical name of the created bucket.
* `bucket_endpoint`: The HTTP(S) endpoint for accessing the bucket.

**Provider Specifics:**
* **AWS**: Requires `region`.
* **Azure**: Requires `subscription_id`, `location`. Sanitizes `bucket_id` by removing non-alphanumeric characters.
* **GCP**: Requires `project_id`, `region`.


