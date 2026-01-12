variable "project_id" {
  description = "The GCP Project ID."
  type        = string
}

variable "region" {
  description = "The GCP Region (e.g. europe-west1)."
  type        = string
}

variable "instance_id" {
  description = "Unique identifier for this resource (e.g. vm-1)."
  type        = string
}

variable "size" {
  description = "small, medium, large."
  type        = string

  validation {
    condition     = contains(["small", "medium", "large"], var.size)
    error_message = "Size must be: small, medium, or large."
  }
}

variable "os" {
  description = "OS Family: debian, ubuntu"
  type        = string
  default     = "debian"
}

variable "disk_size_gb" {
  description = "Boot disk size in GB."
  type        = number
  default     = 10
}

variable "metadata" {
  description = "Arbitrary metadata/labels."
  type        = map(string)
  default     = {}
}
