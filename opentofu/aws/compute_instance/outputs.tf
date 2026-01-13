output "instance_id" {
  value = aws_instance.vm.id
}

output "public_ip" {
  value = aws_instance.vm.public_ip
}

output "private_ip" {
  value = aws_instance.vm.private_ip
}

output "ssh_connection_string" {
  value = var.ssh_public_key == "" ? "ssh -i ${var.instance_id}.pem ${local.ssh_user}@${aws_instance.vm.public_ip}" : "ssh ${local.ssh_user}@${aws_instance.vm.public_ip}"
}
