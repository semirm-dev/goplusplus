variable "project_id" {
  type    = string
  default = "floci-local"
}

variable "region" {
  type    = string
  default = "europe-west3"
}

variable "zone" {
  description = "Zonal cluster; use var.region for a regional control plane"
  type        = string
  default     = "europe-west3-a"
}

variable "cluster_name" {
  type    = string
  default = "platform"
}

variable "emulator" {
  description = "Skip what floci-gcp does not implement (Cloud Router/NAT, KMS key IAM). false for real GCP."
  type        = bool
  default     = true
}

variable "admin_cidr" {
  description = "Only this CIDR reaches the API server, e.g. 203.0.113.7/32"
  type        = string

  validation {
    condition     = var.admin_cidr != "0.0.0.0/0"
    error_message = "admin_cidr must not open the API server to the internet, use $(curl -s ifconfig.me)/32."
  }
}

variable "github_repo" {
  description = "owner/name of the repository whose workflow pushes images"
  type        = string
  default     = "semirm-dev/goplusplus"
}
