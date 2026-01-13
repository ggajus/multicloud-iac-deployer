# Multi-Cloud Infrastructure Deployment Parser

This parser converts JSON deployment configurations into `.tfvars` files for OpenTofu/Terraform deployments across AWS, GCP, and Azure.

## Flow

1. JSON schema validation based on `schema.json`
2. Parse `.json` to `.tfvars` files
3. Output in separated directories based on providers -> services -> versions

## Installation

```bash
pip install -r requirements.txt
```

## Usage

```bash
cd parser # Must run in correct folder
python parser.py <config.json>
```