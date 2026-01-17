output "bucket_name" {
  description = "The name of the bucket."
  value       = google_storage_bucket.bucket.name
}

output "bucket_endpoint" {
  description = "The HTTP endpoint of the bucket."
  value       = "https://storage.googleapis.com/${google_storage_bucket.bucket.name}"
}

output "bucket_url" {
  description = "The gs:// url of the bucket."
  value       = "gs://${google_storage_bucket.bucket.name}"
}

output "bucket_self_link" {
  description = "The URI of the created resource."
  value       = google_storage_bucket.bucket.self_link
}