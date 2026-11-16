terraform {
  required_version = ">= 1.5.0"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 6.0"
    }
  }
}

# null leaves an attribute unset: real endpoints and Application Default Credentials
provider "google" {
  project      = var.project_id
  region       = var.region
  access_token = var.emulator ? "floci-local-dummy-token" : null

  container_custom_endpoint              = var.emulator ? "http://container.localhost:4588/" : null
  compute_custom_endpoint                = var.emulator ? "http://localhost:4588/compute/v1/" : null
  iam_custom_endpoint                    = var.emulator ? "http://localhost:4588/" : null
  cloud_resource_manager_custom_endpoint = var.emulator ? "http://localhost:4588/" : null
  service_usage_custom_endpoint          = var.emulator ? "http://localhost:4588/" : null
  resource_manager_custom_endpoint       = var.emulator ? "http://localhost:4588/" : null
  cloud_billing_custom_endpoint          = var.emulator ? "http://localhost:4588/" : null
  kms_custom_endpoint                    = var.emulator ? "http://localhost:4588/v1/" : null
}
