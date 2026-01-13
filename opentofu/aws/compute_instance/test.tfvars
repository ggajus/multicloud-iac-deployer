region         = "eu-north-1"
instance_id    = "test-aws-vm-01"
size           = "small"
os             = "ubuntu"
disk_size_gb   = 20
metadata = {
  app  = "demo-app"
  tier = "backend"
}

ssh_public_key = ""
