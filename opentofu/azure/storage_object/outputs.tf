output "storage_account_name" {
  value = azurerm_storage_account.account.name
}

output "bucket_name" {
  value = azurerm_storage_account.account.name
}

output "container_name" {
  value = azurerm_storage_container.container.name
}

output "primary_blob_endpoint" {
  value = azurerm_storage_account.account.primary_blob_endpoint
}

output "bucket_endpoint" {
  value = azurerm_storage_account.account.primary_blob_endpoint
}
