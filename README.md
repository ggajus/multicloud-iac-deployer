# Multi-Cloud IaC Provisioner

A protype CLI tool to deploy infrastructure (Compute Instances and Storage Buckets) across AWS, GCP and Azure using a unified JSON configuration and OpenTofu.

## Prerequisites

- **Go** (1.25+)
- **OpenTofu** (installed as `tofu`)
- **Cloud Credentials** see `.env.example`

## Installation

```bash
go build -o deployer ./cmd/deployer
```

## Usage

### 1. Provision Infrastructure

Deploy resources defined in a config file. Example configs can be found in the `examples` folder.

```bash
./deployer deploy <config_file.json>
```

**Example Config (`examples/azure_demo.json`):**
```json
{
  "project_name": "azure-demo-project",
  "provider": "azure",
  "region": "Sweden Central",
  "version": "v1.0.0",
  "services": [
    {
      "type": "compute.instance",
      "instance_id": "az-demo-vm",
      "size": "small",
      "os": "ubuntu",
      "disk_size_gb": 30,
      "metadata": {
        "app": "demo-app",
        "tier": "backend"
      },
      "ssh_public_key": "",
      "allowed_ports": [
        22,
        80,
        443
      ]
    },
    {
      "type": "storage.object",
      "bucket_id": "skycontroldemoazunique",
      "storage_tier": "standard",
      "versioning": false
    }
  ]
}
```

### 2. View Outputs

View connection strings, IPs, and other outputs for an existing deployment.

```bash
./deployer output deployment/<provider>/<project_name>
```

### 3. Destroy Infrastructure

Tear down all resources in a deployment directory.

```bash
./deployer destroy deployment/<provider>/<project_name>
```

### 4. Verify Credentials

Test if your cloud provider environment variables are set correctly.

```bash
./deployer verify-creds
```

### 5. Test Provisioning
Provisioning can be tested using the example json configuration files located in the `examples` folder.
```bash
go test -v -tags=integration ./cmd/deployer 
```

## Project Structure

- `cmd/deployer`: Main application logic.
- `pkg/config`: Configuration parsing and validation.
- `opentofu/`: Terraform/OpenTofu modules for each provider.
- `parser/`: JSON schema and generator configuration.
