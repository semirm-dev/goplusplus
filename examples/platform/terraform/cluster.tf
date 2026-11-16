# Created by the kms/ root, only looked up here
data "google_kms_crypto_key" "gke_secrets_key" {
  name     = "${var.cluster_name}-secrets-key"
  key_ring = "projects/${var.project_id}/locations/${var.region}/keyRings/${var.cluster_name}-keyring"
}

resource "google_container_cluster" "primary" {
  name     = var.cluster_name
  location = var.zone

  network    = google_compute_network.vpc.id
  subnetwork = google_compute_subnetwork.gke_subnet.id

  # No initial_node_count and no node_config: GKE creates no default pool.
  # Nodes come only from google_container_node_pool.workers.
  deletion_protection = false # true for a cluster you keep

  datapath_provider = "ADVANCED_DATAPATH" # Dataplane V2

  release_channel {
    channel = "REGULAR"
  }

  ip_allocation_policy {
    cluster_secondary_range_name  = "k8s-pods"
    services_secondary_range_name = "k8s-services"
  }

  private_cluster_config {
    enable_private_nodes    = true
    enable_private_endpoint = false
    master_ipv4_cidr_block  = "172.16.0.0/28"
  }

  master_authorized_networks_config {
    cidr_blocks {
      cidr_block   = var.admin_cidr
      display_name = "admin"
    }
  }

  workload_identity_config {
    workload_pool = "${var.project_id}.svc.id.goog"
  }

  database_encryption {
    state    = "ENCRYPTED"
    key_name = data.google_kms_crypto_key.gke_secrets_key.id
  }

  gateway_api_config {
    channel = "CHANNEL_STANDARD"
  }

  logging_config {
    enable_components = ["SYSTEM_COMPONENTS", "WORKLOADS"]
  }

  monitoring_config {
    enable_components = ["SYSTEM_COMPONENTS"]
    managed_prometheus {
      enabled = true
    }
  }

  depends_on = [
    google_project_service.apis,
    google_project_iam_member.node_default_role,
  ]
}

# The control plane reaches private nodes only on 443 and 10250 by default.
# The OpenBao injector (8080) and Kyverno (9443) serve their webhooks elsewhere.
resource "google_compute_firewall" "control_plane_webhooks" {
  name          = "${var.cluster_name}-control-plane-webhooks"
  network       = google_compute_network.vpc.id
  direction     = "INGRESS"
  source_ranges = [google_container_cluster.primary.private_cluster_config[0].master_ipv4_cidr_block]

  allow {
    protocol = "tcp"
    ports    = ["8080", "9443"]
  }
}
