# Created by the kms/ root, only looked up here
data "google_kms_crypto_key" "openbao_unseal" {
  name     = "${var.cluster_name}-openbao-unseal"
  key_ring = "projects/${var.project_id}/locations/${var.region}/keyRings/${var.cluster_name}-keyring"
}

resource "google_service_account" "openbao" {
  account_id   = "openbao"
  display_name = "OpenBao auto-unseal"
}

# Wrap and unwrap the root key, on this one key only. floci-gcp answers 404 for key IAM policies.
resource "google_kms_crypto_key_iam_member" "openbao_unseal" {
  count = var.emulator ? 0 : 1

  crypto_key_id = data.google_kms_crypto_key.openbao_unseal.id
  role          = "roles/cloudkms.cryptoKeyEncrypterDecrypter"
  member        = "serviceAccount:${google_service_account.openbao.email}"
}

# The gcpckms seal reads the key's metadata on startup
resource "google_kms_crypto_key_iam_member" "openbao_viewer" {
  count = var.emulator ? 0 : 1

  crypto_key_id = data.google_kms_crypto_key.openbao_unseal.id
  role          = "roles/cloudkms.viewer"
  member        = "serviceAccount:${google_service_account.openbao.email}"
}

# KSA openbao/openbao may act as this GSA. No key is ever created.
resource "google_service_account_iam_member" "openbao_wi" {
  service_account_id = google_service_account.openbao.name
  role               = "roles/iam.workloadIdentityUser"
  member             = "serviceAccount:${var.project_id}.svc.id.goog[openbao/openbao]"
}

output "openbao_service_account" {
  value = google_service_account.openbao.email
}
