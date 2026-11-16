variable "project_id" {
  type    = string
  default = "floci-local"
}

variable "region" {
  type    = string
  default = "europe-west3"
}

variable "cluster_name" {
  type    = string
  default = "platform"
}

variable "emulator" {
  description = "floci-gcp does not implement KMS key IAM. false for real GCP."
  type        = bool
  default     = true
}
