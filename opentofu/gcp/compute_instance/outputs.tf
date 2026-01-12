output "instance_self_link" {
  value = google_compute_instance.vm.self_link
}

output "internal_ip" {
  value = google_compute_instance.vm.network_interface.0.network_ip
}

output "external_ip" {
  value = length(google_compute_instance.vm.network_interface.0.access_config) > 0 ? google_compute_instance.vm.network_interface.0.access_config.0.nat_ip : "None"
}
