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

# terraform_data is a built-in resource (no provider download needed). Setting
# `input` to `timestamp()` makes the input value differ on every plan, so the
# resource is never idempotent — Plan always reports a change. This module
# exists purely to exercise the AssertIdempotent failure path.
resource "terraform_data" "always_drifts" {
  input = timestamp()
}

resource "aws_s3_bucket" "anchor" {
  bucket = var.bucket_name
}

output "drift_input" {
  description = "The current drift-input value (changes on every plan)."
  value       = terraform_data.always_drifts.input
}
