variable "region" {
  description = "AWS Region (e.g., us-east-1, eu-central-1)."
  type        = string
}

variable "instance_id" {
  description = "Unique identifier (used for Name tags)."
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
  description = "OS Family: ubuntu, debian."
  type        = string
  default     = "ubuntu"

  validation {
    condition     = contains(["ubuntu", "debian"], var.os)
    error_message = "OS must be: ubuntu or debian."
  }
}

variable "disk_size_gb" {
  description = "Size of the root volume in GB."
  type        = number
  default     = 20
}

variable "metadata" {
  description = "Tags to assign to resources."
  type        = map(string)
  default     = {}
}

variable "ssh_public_key" {
  description = "SSH Public Key string. If empty, one will be generated."
  type        = string
  default     = ""
}

variable "allowed_ports" {
  description = "List of ports to allow ingress traffic on."
  type        = list(number)
  default     = []
}