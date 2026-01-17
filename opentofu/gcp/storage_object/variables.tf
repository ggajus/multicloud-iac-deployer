variable "project_id" {
  description = "The GCP Project ID."
  type        = string
}

variable "region" {
  description = "The GCP Region (e.g. europe-west1)."
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
