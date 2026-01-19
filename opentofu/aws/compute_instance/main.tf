locals {
  size_map = {
    "small"  = "t3.micro"       # 2 vCPU, 1 GB RAM
    "medium" = "t3.small"       # 2 vCPU, 4 GB RAM
    "large"  = "m7i-flex.large" # 2 vCPU, 8 GB RAM
  }

  ami_id = var.os == "ubuntu" ? data.aws_ami.ubuntu.id : data.aws_ami.debian.id

  ssh_user = var.os == "ubuntu" ? "ubuntu" : "admin"
}

data "aws_ami" "ubuntu" {
  most_recent = true
  owners      = ["099720109477"] 
  filter {
    name   = "name"
    values = ["ubuntu/images/hvm-ssd/ubuntu-jammy-22.04-amd64-server-*"]
  }
}

data "aws_ami" "debian" {
  most_recent = true
  owners      = ["136693071363"] 
  filter {
    name   = "name"
    values = ["debian-12-amd64-*"]
  }
}

resource "tls_private_key" "gen" {
  count     = var.ssh_public_key == "" ? 1 : 0
  algorithm = "RSA"
  rsa_bits  = 4096
}

resource "aws_key_pair" "auth" {
  key_name   = "${var.instance_id}-key"
  public_key = var.ssh_public_key != "" ? var.ssh_public_key : tls_private_key.gen[0].public_key_openssh
}

resource "aws_vpc" "vpc" {
  cidr_block = "10.0.0.0/16"
  tags       = merge(var.metadata, { Name = "${var.instance_id}-vpc" })
}

resource "aws_internet_gateway" "igw" {
  vpc_id = aws_vpc.vpc.id
  tags   = { Name = "${var.instance_id}-igw" }
}

resource "aws_subnet" "subnet" {
  vpc_id                  = aws_vpc.vpc.id
  cidr_block              = "10.0.1.0/24"
  map_public_ip_on_launch = true # Auto-assign public IP
  tags                    = { Name = "${var.instance_id}-subnet" }
}

resource "aws_route_table" "rt" {
  vpc_id = aws_vpc.vpc.id
  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.igw.id
  }
}

resource "aws_route_table_association" "a" {
  subnet_id      = aws_subnet.subnet.id
  route_table_id = aws_route_table.rt.id
}

resource "aws_security_group" "sg" {
  name   = "${var.instance_id}-sg"
  vpc_id = aws_vpc.vpc.id

  ingress {
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  dynamic "ingress" {
    for_each = var.allowed_ports
    content {
      from_port   = ingress.value
      to_port     = ingress.value
      protocol    = "tcp"
      cidr_blocks = ["0.0.0.0/0"]
    }
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_instance" "vm" {
  ami           = local.ami_id
  instance_type = local.size_map[var.size]
  key_name      = aws_key_pair.auth.key_name

  subnet_id                   = aws_subnet.subnet.id
  vpc_security_group_ids      = [aws_security_group.sg.id]
  associate_public_ip_address = true

  root_block_device {
    volume_size = var.disk_size_gb
    volume_type = "gp3" # General Purpose SSD
  }

  tags = merge(var.metadata, {
    Name       = var.instance_id
    managed_by = "sky-control"
  })
}

resource "local_file" "private_key" {
  count           = var.ssh_public_key == "" ? 1 : 0
  content         = tls_private_key.gen[0].private_key_pem
  filename        = "${path.module}/${var.instance_id}.pem"
  file_permission = "0600"
}
