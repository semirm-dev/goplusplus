terraform {
  required_version = ">= 1.5.0"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 6.0"
    }
  }
}

provider "google" {
  project      = var.project_id
  region       = var.region
  access_token = var.emulator ? "floci-local-dummy-token" : null

  cloud_resource_manager_custom_endpoint = var.emulator ? "http://localhost:4588/" : null
  service_usage_custom_endpoint          = var.emulator ? "http://localhost:4588/" : null
  kms_custom_endpoint                    = var.emulator ? "http://localhost:4588/v1/" : null
}
