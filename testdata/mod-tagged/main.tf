terraform {
  required_version = ">= 1.5"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 5.0"
    }
  }
}

provider "aws" {
  region = "us-east-1"
}

resource "aws_s3_bucket" "primary" {
  bucket = var.bucket_name
  tags   = var.tags
}

resource "aws_s3_bucket" "secondary" {
  bucket = "${var.bucket_name}-2"
  tags   = var.tags
}

output "primary_arn" {
  description = "ARN of the primary tagged bucket."
  value       = aws_s3_bucket.primary.arn
}

output "secondary_arn" {
  description = "ARN of the secondary tagged bucket."
  value       = aws_s3_bucket.secondary.arn
}
