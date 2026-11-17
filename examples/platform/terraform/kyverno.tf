# Kyverno reads signatures and attestations from Artifact Registry to verify images.
# Without an identity of its own it gets none, and every api pod fails on Enforce.
resource "google_service_account" "kyverno" {
  account_id   = "kyverno"
  display_name = "Kyverno image verification"
}

# reader on this one repository, not on the project
resource "google_artifact_registry_repository_iam_member" "kyverno_reader" {
  project    = var.project_id
  location   = google_artifact_registry_repository.apps.location
  repository = google_artifact_registry_repository.apps.name
  role       = "roles/artifactregistry.reader"
  member     = google_service_account.kyverno.member
}

# KSA kyverno/kyverno-admission-controller may act as this GSA
resource "google_service_account_iam_member" "kyverno_wi" {
  service_account_id = google_service_account.kyverno.name
  role               = "roles/iam.workloadIdentityUser"
  member             = "serviceAccount:${var.project_id}.svc.id.goog[kyverno/kyverno-admission-controller]"
}
