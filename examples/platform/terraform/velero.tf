# Multi-region: a backup must outlive the region the cluster runs in
resource "google_storage_bucket" "velero" {
  name                        = "${var.project_id}-velero"
  location                    = "EU"
  uniform_bucket_level_access = true
  public_access_prevention    = "enforced"
  force_destroy               = true # dev cluster, destroyed daily; false for a cluster you keep

  # Velero must delete expired backups, so versioning and soft delete are what survive a deletion
  versioning {
    enabled = true
  }

  soft_delete_policy {
    retention_duration_seconds = 604800 # 7 days
  }

  lifecycle_rule {
    condition {
      days_since_noncurrent_time = 30
      with_state                 = "ARCHIVED"
    }
    action {
      type = "Delete"
    }
  }
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
