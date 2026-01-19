locals {
  size_map = {
    "small"  = "e2-micro"
    "medium" = "e2-medium"
    "large"  = "e2-standard-2"
  }

  os_map = {
    "debian" = "debian-cloud/debian-11"
    "ubuntu" = "ubuntu-os-cloud/ubuntu-2204-lts"
  }
}

resource "google_compute_instance" "vm" {
  name = var.instance_id

  machine_type = local.size_map[var.size]

  zone    = var.zone
  project = var.project_id

  allow_stopping_for_update = true

  boot_disk {
    initialize_params {
      image  = local.os_map[var.os]
      size   = var.disk_size_gb
      labels = var.metadata
    }
  }

  network_interface {
    network = "default"
    access_config {
    }
  }

  tags = ["sky-control-firewall"]

  labels = merge(var.metadata, {
    managed_by = "sky-control"
  })

  service_account {
    scopes = ["cloud-platform"]
  }
}

resource "google_compute_firewall" "fw" {
  name    = "${var.instance_id}-fw"
  network = "default"
  project = var.project_id

  allow {
    protocol = "tcp"
    ports    = concat(["22"], [for p in var.allowed_ports : tostring(p)])
  }

  source_ranges = ["0.0.0.0/0"]
  target_tags   = ["sky-control-firewall"]
}
