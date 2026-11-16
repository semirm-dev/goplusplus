# Own root: applied once, never destroyed. Key rings can't be deleted and their names stay taken.
resource "google_project_service" "apis" {
  for_each = toset([
    "cloudkms.googleapis.com",
    "container.googleapis.com", # creates the GKE service agent granted below
  ])

  service            = each.key
  disable_on_destroy = false
}

resource "google_kms_key_ring" "gke_keyring" {
  name       = "${var.cluster_name}-keyring"
  location   = var.region # must be the cluster's region
  depends_on = [google_project_service.apis]
}

resource "google_kms_crypto_key" "gke_secrets_key" {
  name            = "${var.cluster_name}-secrets-key"
  key_ring        = google_kms_key_ring.gke_keyring.id
  rotation_period = "7776000s" # 90 days

  lifecycle {
    prevent_destroy = true
  }
}

data "google_project" "current" {}

# floci-gcp answers 404 for key IAM policies
resource "google_kms_crypto_key_iam_member" "gke_kms_encrypter" {
  count = var.emulator ? 0 : 1

  crypto_key_id = google_kms_crypto_key.gke_secrets_key.id
  role          = "roles/cloudkms.cryptoKeyEncrypterDecrypter"
  member        = "serviceAccount:service-${data.google_project.current.number}@container-engine-robot.iam.gserviceaccount.com"
}

output "gke_secrets_key_id" {
  value = google_kms_crypto_key.gke_secrets_key.id
}
