#!/usr/bin/env python3
"""
Multi-Cloud Infrastructure Deployment Parser

This script parses JSON deployment configurations and generates .tfvars files
for OpenTofu/Terraform deployments across AWS, GCP, and Azure.
"""

import json
import os
import sys
from datetime import datetime
from pathlib import Path
from typing import Dict, List, Any, Optional
import jsonschema


class DeploymentParser:
    def __init__(self, schema_path: str):
        self.schema = self._load_schema(schema_path=Path(schema_path))

    def _load_schema(self, schema_path) -> Dict[str, Any]:
        try:
            with open(schema_path, 'r') as f:
                return json.load(f)
        except Exception as e:
            print(f"Error loading schema: {e}")
            sys.exit(1)

    def validate_config(self, config: Dict[str, Any]) -> tuple:
        try:
            jsonschema.validate(instance=config, schema=self.schema)
        except Exception as e:
            return False, str(e)

        # Specific validation for different providers
        provider = config.get("provider")
        services = config.get("services", [])

        # Validate specific GCP
        if provider == "gcp":
            for service in services:
                if service.get("type") == "compute.instance" and not service.get("project_id"):
                    return False, "GCP compute.instance requires 'project_id' in service configuration"
        return True, ""

    def _generate_version_dir(self, config: Dict[str, Any]) -> Path:
        # Version set in config or timestamp as default
        version = config.get("version")
        if not version:
            timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
            version = f"deployment_{timestamp}"
        return Path(version)

    def _format_tfvars_value(self, value: Any) -> str:
        # Format a value for .tfvars file
        if isinstance(value, str):
            return f'"{value}"'
        elif isinstance(value, (int, float)):
            return str(value)
        elif isinstance(value, bool):
            return "true" if value else "false"
        elif isinstance(value, dict):
            items = [f'  {k} = {self._format_tfvars_value(v)}' for k, v in value.items()]
            return "{\n" + "\n".join(items) + "\n}"
        elif isinstance(value, list):
            items = [self._format_tfvars_value(item) for item in value]
            return "[" + ", ".join(items) + "]"
        else:
            return str(value)

    def _generate_aws_tfvars(self, config: Dict[str, Any], service: Dict[str, Any]) -> str:
        # Generate AWS .tfvars content
        lines = []
        lines.append(f'region         = "{config["region"]}"')
        lines.append(f'instance_id    = "{service["instance_id"]}"')
        lines.append(f'size           = "{service["size"]}"')
        lines.append(f'os             = "{service["os"]}"')

        if "disk_size_gb" in service:
            lines.append(f'disk_size_gb   = {service["disk_size_gb"]}')

        if "metadata" in service and service["metadata"]:
            metadata_str = self._format_tfvars_value(service["metadata"])
            lines.append(f'metadata = {metadata_str}')

        if "ssh_public_key" in service:
            lines.append(f'ssh_public_key = "{service.get("ssh_public_key", "")}"')
        else:
            lines.append('ssh_public_key = ""')

        return "\n".join(lines) + "\n"

    def _generate_gcp_tfvars(self, config: Dict[str, Any], service: Dict[str, Any]) -> str:
        # Generate GCP .tfvars content
        lines = []
        if "project_id" in service:
            lines.append(f'project_id  = "{service["project_id"]}"')
        lines.append(f'region      = "{config["region"]}"')
        lines.append(f'instance_id = "{service["instance_id"]}"')
        lines.append(f'size        = "{service["size"]}"')
        lines.append(f'os          = "{service["os"]}"')

        if "disk_size_gb" in service:
            lines.append(f'disk_size_gb = {service["disk_size_gb"]}')

        if "metadata" in service and service["metadata"]:
            metadata_str = self._format_tfvars_value(service["metadata"])
            lines.append(f'metadata = {metadata_str}')

        return "\n".join(lines) + "\n"

    def _generate_azure_tfvars(self, config: Dict[str, Any], service: Dict[str, Any]) -> str:
        # Generate Azure .tfvars content
        lines = []
        # Azure uses "location" instead of "region"
        lines.append(f'location    = "{config["region"]}"')
        lines.append(f'instance_id = "{service["instance_id"]}"')
        lines.append(f'size        = "{service["size"]}"')
        lines.append(f'os          = "{service["os"]}"')

        if "disk_size_gb" in service:
            lines.append(f'disk_size_gb = {service["disk_size_gb"]}')

        if "metadata" in service and service["metadata"]:
            metadata_str = self._format_tfvars_value(service["metadata"])
            lines.append(f'metadata = {metadata_str}')

        if "admin_username" in service:
            lines.append(f'admin_username = "{service["admin_username"]}"')

        if "ssh_public_key" in service:
            lines.append(f'ssh_public_key = "{service.get("ssh_public_key", "")}"')

        return "\n".join(lines) + "\n"

    def _get_service_folder_name(self, service_type: str) -> str:
        # Get folder name for service type
        if service_type == "compute.instance":
            return "compute_instance"
        elif service_type == "storage.object":
            return "storage_object"
        else:
            return service_type.replace(".", "_")

    def parse(self, config_path: str) -> bool:
        # Load configuration
        try:
            with open(config_path, 'r') as f:
                config = json.load(f)
        except Exception as e:
            print(f"[ERROR] Error loading configuration: {e}")
            return False

        # Validate configuration
        is_valid, error_msg = self.validate_config(config)
        if not is_valid:
            print(f"[ERROR] Validation failed: {error_msg}")
            return False

        print(f"✓ Configuration validated successfully")

        # Generate version directory
        version_dir = self._generate_version_dir(config)
        provider = config["provider"]
        services = config["services"]

        # Create output directory structure
        output_dir = Path("../opentofu") / provider / version_dir

        # Process each service
        for service in services:
            service_type = service["type"]
            service_folder = self._get_service_folder_name(service_type)

            # Create service directory
            service_dir = output_dir / service_folder
            service_dir.mkdir(parents=True, exist_ok=True)

            # Generate .tfvars content based on provider
            if provider == "aws":
                tfvars_content = self._generate_aws_tfvars(config, service)
            elif provider == "gcp":
                tfvars_content = self._generate_gcp_tfvars(config, service)
            elif provider == "azure":
                tfvars_content = self._generate_azure_tfvars(config, service)
            else:
                print(f"[ERROR] Unknown provider {provider}")
                return False

            # Write .tfvars file
            tfvars_file = service_dir / f'opentofu.tfvars'
            with open(tfvars_file, 'w') as f:
                f.write(tfvars_content)

            print(f"✓ Generated: {tfvars_file}")

        print(f"\n✓ Successfully parsed configuration")
        print(f"  Output directory: {output_dir}")
        return True

SCHEMA_PATH = Path(__file__).parent / "schema.json"
def main():
    """Main entry point."""
    if len(sys.argv) < 1:
        print("Usage: python parser.py <config.json>")
        sys.exit(1)

    config_path = sys.argv[1]

    # Get the directory where the script is located
    parser = DeploymentParser(schema_path=str(SCHEMA_PATH))
    success = parser.parse(config_path)
    
    return success


if __name__ == "__main__":
    main()
