resource "google_storage_bucket" "velero" {
  name                        = "${var.project_id}-velero"
  location                    = var.region
  uniform_bucket_level_access = true
  public_access_prevention    = "enforced"
}

resource "google_service_account" "velero" {
  account_id   = "velero"
  display_name = "Velero"
}

resource "google_storage_bucket_iam_member" "velero" {
  bucket = google_storage_bucket.velero.name
  role   = "roles/storage.objectAdmin"
  member = "serviceAccount:${google_service_account.velero.email}"
}

# KSA velero/velero may act as this GSA. No key is ever created.
resource "google_service_account_iam_member" "velero_wi" {
  service_account_id = google_service_account.velero.name
  role               = "roles/iam.workloadIdentityUser"
  member             = "serviceAccount:${var.project_id}.svc.id.goog[velero/velero]"
}

# signBlob on itself, for the signed URLs behind `velero backup logs/download`
resource "google_service_account_iam_member" "velero_sign" {
  service_account_id = google_service_account.velero.name
  role               = "roles/iam.serviceAccountTokenCreator"
  member             = "serviceAccount:${google_service_account.velero.email}"
}
