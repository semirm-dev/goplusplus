resource "google_project_service" "apis" {
  for_each = toset([
    "compute.googleapis.com",
    "container.googleapis.com",
    "iam.googleapis.com",
    "artifactregistry.googleapis.com",
  ])

  service            = each.key
  disable_on_destroy = false
}

resource "google_compute_network" "vpc" {
  name                    = "${var.cluster_name}-vpc"
  routing_mode            = "REGIONAL"
  auto_create_subnetworks = false
  depends_on              = [google_project_service.apis]
}

resource "google_compute_subnetwork" "gke_subnet" {
  name                     = "${var.cluster_name}-private-subnet"
  ip_cidr_range            = "10.0.0.0/20" # nodes
  region                   = var.region
  network                  = google_compute_network.vpc.id
  private_ip_google_access = true # Artifact Registry and Google APIs without NAT

  secondary_ip_range {
    range_name    = "k8s-pods"
    ip_cidr_range = "10.16.0.0/14" # a /24 per node, room for 1024 nodes
  }

  secondary_ip_range {
    range_name    = "k8s-services"
    ip_cidr_range = "10.20.0.0/20"
  }
}

# Outbound internet for private nodes. floci-gcp answers 501 for both.
resource "google_compute_router" "router" {
  count = var.emulator ? 0 : 1

  name    = "${var.cluster_name}-router"
  region  = var.region
  network = google_compute_network.vpc.id
}

resource "google_compute_router_nat" "nat" {
  count = var.emulator ? 0 : 1

  name                               = "${var.cluster_name}-nat"
  router                             = google_compute_router.router[0].name
  region                             = var.region
  nat_ip_allocate_option             = "AUTO_ONLY"
  source_subnetwork_ip_ranges_to_nat = "ALL_SUBNETWORKS_ALL_IP_RANGES"
}
