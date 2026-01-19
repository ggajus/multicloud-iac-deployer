variable "region" {
  description = "Azure Region (e.g., West Europe, East US)."
  type        = string
}

variable "bucket_id" {
  description = "Unique identifier (used for storage account name). Must be 3-24 chars, lowercase alphanumeric."
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
  description = "Tags to assign to resources."
  type        = map(string)
  default     = {}
}
