resource "aws_s3_bucket" "bucket" {
  bucket = local.bucket_name

  # Force destroy allows deletion even if not empty (dev/prototype safety)
  force_destroy = true

  tags = merge(var.metadata, {
    managed_by = "sky-control"
  })
}

resource "aws_s3_bucket_versioning" "versioning" {
  bucket = aws_s3_bucket.bucket.id
  versioning_configuration {
    status = var.versioning ? "Enabled" : "Suspended"
  }
}

resource "aws_s3_bucket_lifecycle_configuration" "tiering" {
  # We only attach lifecycle rules if the requested tier is NOT standard.
  # Standard is the default behavior.
  count  = var.storage_tier == "standard" ? 0 : 1
  bucket = aws_s3_bucket.bucket.id

  rule {
    id     = "enforce-tier-${var.storage_tier}"
    status = "Enabled"

    filter {
      prefix = ""
    }

    transition {
      days          = lookup(local.days_map, var.storage_tier, 0)
      storage_class = lookup(local.tier_map, var.storage_tier, "STANDARD")
    }
  }
}

locals {
  bucket_name = lower(var.bucket_id)
  tier_map = {
    # standard is handled by count=0 above
    "infrequent" = "STANDARD_IA"
    "cold"       = "GLACIER"
    "archive"    = "DEEP_ARCHIVE"
  }

  days_map = {
    "infrequent" = 30 # AWS requires min 30 days for STANDARD_IA
    "cold"       = 0
    "archive"    = 0
  }
}