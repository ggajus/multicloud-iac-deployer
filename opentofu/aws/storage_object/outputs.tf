output "bucket_name" {
  description = "The name of the bucket."
  value       = aws_s3_bucket.bucket.id
}

output "bucket_endpoint" {
  description = "The regional domain name of the bucket."
  value       = aws_s3_bucket.bucket.bucket_regional_domain_name
}

output "bucket_arn" {
  description = "The ARN of the bucket."
  value       = aws_s3_bucket.bucket.arn
}