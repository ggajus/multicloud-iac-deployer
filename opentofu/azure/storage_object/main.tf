locals {
  # Map abstract tiers to Azure Access Tiers
  tier_map = {
    "standard"   = "Hot"
    "infrequent" = "Cool"
    "cold"       = "Cold"
    "archive"    = "Cool" # Archive is blob-level, account stays Cool
  }

  # Storage account names must be globally unique and 3-24 characters
  # We'll sanitize the input slightly by removing non-alphanumeric chars
  # Note: In a real system, the orchestrator should handle uniqueness.
  storage_account_name = lower(replace(var.bucket_id, "/[^a-zA-Z0-9]/", ""))
}

resource "azurerm_resource_group" "rg" {
  name     = "${var.bucket_id}-rg"
  location = var.region
  tags     = var.metadata
}

resource "azurerm_storage_account" "account" {
  name                     = local.storage_account_name
  resource_group_name      = azurerm_resource_group.rg.name
  location                 = azurerm_resource_group.rg.location
  account_tier             = "Standard"
  account_replication_type = "LRS"
  access_tier              = local.tier_map[var.storage_tier]

  blob_properties {
    versioning_enabled = var.versioning
  }

  tags = merge(var.metadata, {
    managed_by = "sky-control"
  })
}

resource "azurerm_storage_container" "container" {
  name                  = "content"
  storage_account_id    = azurerm_storage_account.account.id
  container_access_type = "private"
}
