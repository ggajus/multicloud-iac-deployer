locals {
  size_map = {
    "small"  = "Standard_B2ats_v2" # 2 vCPU, 1 GB RAM
    "medium" = "Standard_B2als_v2" # 2 vCPU, 4 GB RAM
    "large"  = "Standard_B2as_v2"  # 2 vCPU, 8 GB RAM
  }

  os_map = {
    "ubuntu" = {
      publisher = "Canonical"
      offer     = "0001-com-ubuntu-server-jammy"
      sku       = "22_04-lts"
      version   = "latest"
    }
    "debian" = {
      publisher = "Debian"
      offer     = "debian-11"
      sku       = "11"
      version   = "latest"
    }
  }
}

resource "tls_private_key" "example" {
  count     = var.ssh_public_key == "" ? 1 : 0
  algorithm = "RSA"
  rsa_bits  = 4096
}

resource "azurerm_resource_group" "rg" {
  name     = "${var.instance_id}-rg"
  location = var.region
  tags     = var.metadata
}

resource "azurerm_virtual_network" "vnet" {
  name                = "${var.instance_id}-vnet"
  address_space       = ["10.0.0.0/16"]
  location            = azurerm_resource_group.rg.location
  resource_group_name = azurerm_resource_group.rg.name
}

resource "azurerm_subnet" "subnet" {
  name                 = "internal"
  resource_group_name  = azurerm_resource_group.rg.name
  virtual_network_name = azurerm_virtual_network.vnet.name
  address_prefixes     = ["10.0.1.0/24"]
}

resource "azurerm_public_ip" "pip" {
  name                = "${var.instance_id}-pip"
  resource_group_name = azurerm_resource_group.rg.name
  location            = azurerm_resource_group.rg.location
  allocation_method   = "Static" # Static so we can output it immediately
  sku                 = "Standard"
}

resource "azurerm_network_security_group" "nsg" {
  name                = "${var.instance_id}-nsg"
  location            = azurerm_resource_group.rg.location
  resource_group_name = azurerm_resource_group.rg.name

  security_rule {
    name                       = "SSH"
    priority                   = 1001
    direction                  = "Inbound"
    access                     = "Allow"
    protocol                   = "Tcp"
    source_port_range          = "*"
    destination_port_range     = "22"
    source_address_prefix      = "*"
    destination_address_prefix = "*"
  }

  dynamic "security_rule" {
    for_each = var.allowed_ports
    content {
      name                       = "Allow-${security_rule.value}"
      priority                   = 1002 + security_rule.key
      direction                  = "Inbound"
      access                     = "Allow"
      protocol                   = "Tcp"
      source_port_range          = "*"
      destination_port_range     = security_rule.value
      source_address_prefix      = "*"
      destination_address_prefix = "*"
    }
  }
}

resource "azurerm_network_interface" "nic" {
  name                = "${var.instance_id}-nic"
  location            = azurerm_resource_group.rg.location
  resource_group_name = azurerm_resource_group.rg.name

  ip_configuration {
    name                          = "internal"
    subnet_id                     = azurerm_subnet.subnet.id
    private_ip_address_allocation = "Dynamic"
    public_ip_address_id          = azurerm_public_ip.pip.id
  }
}

resource "azurerm_network_interface_security_group_association" "nsg_assoc" {
  network_interface_id      = azurerm_network_interface.nic.id
  network_security_group_id = azurerm_network_security_group.nsg.id
}

resource "azurerm_linux_virtual_machine" "vm" {
  name                = var.instance_id
  resource_group_name = azurerm_resource_group.rg.name
  location            = azurerm_resource_group.rg.location
  size                = local.size_map[var.size]
  admin_username      = var.admin_username

  network_interface_ids = [
    azurerm_network_interface.nic.id,
  ]

  admin_ssh_key {
    username = var.admin_username
    # Use provided key OR generated key
    public_key = var.ssh_public_key != "" ? var.ssh_public_key : tls_private_key.example[0].public_key_openssh
  }

  os_disk {
    caching              = "ReadWrite"
    storage_account_type = "Standard_LRS" # Cheapest disk type
    disk_size_gb         = var.disk_size_gb
  }

  source_image_reference {
    publisher = local.os_map[var.os].publisher
    offer     = local.os_map[var.os].offer
    sku       = local.os_map[var.os].sku
    version   = local.os_map[var.os].version
  }

  tags = merge(var.metadata, {
    managed_by = "sky-control"
  })
}

resource "local_file" "private_key" {
  count           = var.ssh_public_key == "" ? 1 : 0
  content         = tls_private_key.example[0].private_key_pem
  filename        = "${path.module}/${var.instance_id}.pem"
  file_permission = "0600"
}
