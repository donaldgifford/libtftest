variable "bucket_name" {
  description = "Base name for the tagged S3 buckets."
  type        = string
}

variable "tags" {
  description = "Tag map applied to every bucket in the module."
  type        = map(string)
  default     = {}
}
