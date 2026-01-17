variable "region" {
  description = "AWS Region (e.g., us-east-1, eu-central-1)."
  type        = string
}

variable "bucket_id" {
  description = "Globally unique name for the bucket."
  type        = string
}

variable "storage_tier" {
  description = "Performance tier: standard, infrequent, cold, archive."
  type        = string
  default     = "standard"

  validation {
    condition     = contains(["standard", "infrequent", "cold", "archive"], var.storage_tier)
    error_message = "Storage tier must be: standard, infrequent, cold, or archive."
  }
}

variable "versioning" {
  description = "Enable object versioning."
  type        = bool
  default     = false
}

variable "metadata" {
  description = "Arbitrary metadata/labels."
  type        = map(string)
  default     = {}
}
