# Multi-Cloud Infrastructure Deployment Parser

This parser converts JSON deployment configurations into `.tfvars` files for OpenTofu/Terraform deployments across AWS, GCP, and Azure.

## Flow

1. JSON schema validation based on `schema.json`
2. Parse `.json` to `.tfvars` files
3. Separate output directories based on version

## Installation

```bash
pip install -r requirements.txt
```

## Usage

*Must run in `/parser` folder

```bash
cd parser
python parser.py config.json
```

## JSON Configuration Schema

### Required Fields

- `provider`: Cloud provider (`aws`, `gcp`, or `azure`)
- `region`: Cloud region (e.g., `us-east-1`, `europe-west1`, `West Europe`)
- `services`: Array of service configurations

### Optional Fields

- `version`: Version identifier (if not provided, timestamp-based version will be generated)

### Service Configuration

#### compute.instance 
* Required fields
    - `type`: Must be `"compute.instance"`
    - `instance_id`: Unique identifier for the resource
    - `size`: Instance size (`small`, `medium`, or `large`)
    - `os`: Operating system (`ubuntu` or `debian`)

* Optional fields
    - `disk_size_gb`: Disk size in GB (integer)
    - `metadata`: Key-value pairs for tags/metadata
    - `ssh_public_key`: SSH public key string
    - `project_id`: GCP Project ID (required for GCP)
    - `admin_username`: Admin username (for Azure)

#### storage.object
- `type`: Must be `"storage.object"`
- TBD