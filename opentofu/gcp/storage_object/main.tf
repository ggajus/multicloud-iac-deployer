locals {
  bucket_name = lower(var.bucket_id)
  tier_map = {
    "standard"   = "STANDARD"
    "infrequent" = "NEARLINE"
    "cold"       = "COLDLINE"
    "archive"    = "ARCHIVE"
  }
}

resource "google_storage_bucket" "bucket" {
  name          = local.bucket_name
  location      = regex("^[a-z]+-[a-z]+[0-9]+", var.region)
  project       = var.project_id
  storage_class = local.tier_map[var.storage_tier]

  uniform_bucket_level_access = true
  force_destroy = true

  versioning {
    enabled = var.versioning
  }

  labels = merge(var.metadata, {
    managed_by = "sky-control"
  })
}
