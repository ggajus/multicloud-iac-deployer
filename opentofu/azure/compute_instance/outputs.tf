output "instance_id" {
  value = azurerm_linux_virtual_machine.vm.id
}

output "public_ip" {
  value = azurerm_public_ip.pip.ip_address
}

output "private_ip" {
  value = azurerm_linux_virtual_machine.vm.private_ip_address
}

output "ssh_connection_string" {
  value = var.ssh_public_key == "" ? "ssh -i ${var.instance_id}.pem ${var.admin_username}@${azurerm_public_ip.pip.ip_address}" : "ssh ${var.admin_username}@${azurerm_public_ip.pip.ip_address}"
}
