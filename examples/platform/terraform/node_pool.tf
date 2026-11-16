resource "google_container_node_pool" "workers" {
  name     = "${var.cluster_name}-workers"
  location = var.zone
  cluster  = google_container_cluster.primary.name

  # initial_node_count, not node_count: node_count fights the autoscaler on every apply
  initial_node_count = 1

  autoscaling {
    min_node_count  = 1
    max_node_count  = 4
    location_policy = "BALANCED"
  }

  management {
    auto_repair  = true
    auto_upgrade = true
  }

  upgrade_settings {
    max_surge       = 1
    max_unavailable = 0
  }

  node_config {
    machine_type    = "e2-medium"
    disk_size_gb    = 50
    disk_type       = "pd-balanced"
    service_account = google_service_account.gke_nodes.email
    oauth_scopes    = ["https://www.googleapis.com/auth/cloud-platform"]

    workload_metadata_config {
      mode = "GKE_METADATA"
    }

    shielded_instance_config {
      enable_secure_boot          = true
      enable_integrity_monitoring = true
    }
  }
}
