variable "location" {
  description = "Azure Region (e.g., West Europe, East US)."
  type        = string
}

variable "instance_id" {
  description = "Unique identifier (used for VM name, DNS label, etc)."
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
  description = "ubuntu, debian."
  type        = string
  default     = "ubuntu"

  validation {
    condition     = contains(["ubuntu", "debian"], var.os)
    error_message = "OS must be: ubuntu or debian."
  }
}

variable "disk_size_gb" {
  description = "Size of the OS disk in GB."
  type        = number
  default     = 30 # Azure standard minimum is often 30
}

variable "metadata" {
  description = "Tags to assign to resources."
  type        = map(string)
  default     = {}
}

variable "admin_username" {
  description = "Admin username for the VM."
  type        = string
  default     = "azureuser"
}

variable "ssh_public_key" {
  description = "SSH Public Key string. If empty, one will be generated."
  type        = string
  default     = ""
}
