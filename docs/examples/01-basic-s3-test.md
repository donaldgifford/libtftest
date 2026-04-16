# Basic S3 Module Test

The simplest libtftest usage: test an S3 bucket module with a single test
function. No `TestMain` needed -- libtftest starts and stops a container per
test.

## Module Under Test

```hcl
# modules/s3-bucket/main.tf
resource "aws_s3_bucket" "this" {
  bucket = var.bucket_name
}

resource "aws_s3_bucket_versioning" "this" {
  bucket = aws_s3_bucket.this.id
  versioning_configuration {
    status = "Enabled"
  }
}

# modules/s3-bucket/variables.tf
variable "bucket_name" {
  type = string
}

# modules/s3-bucket/outputs.tf
output "bucket_id" {
  value = aws_s3_bucket.this.id
}
```

## Test File

```go
//go:build integration

package test

import (
    "testing"

    "github.com/donaldgifford/libtftest"
    "github.com/donaldgifford/libtftest/assert"
    "github.com/donaldgifford/libtftest/localstack"
)

func TestS3Bucket_CreatesWithVersioning(t *testing.T) {
    t.Parallel()

    tc := libtftest.New(t, &libtftest.Options{
        Edition:   localstack.EditionCommunity,
        ModuleDir: "../../modules/s3-bucket",
    })
    tc.SetVar("bucket_name", tc.Prefix()+"-test-bucket")

    tc.Apply()

    bucket := tc.Output("bucket_id")

    // Assert the bucket exists and has versioning enabled.
    assert.S3.BucketExists(t, tc.AWS(), bucket)
    assert.S3.BucketHasVersioning(t, tc.AWS(), bucket)
}
```

## What Happens

1. `libtftest.New` starts a LocalStack Community container
2. Copies the module to a scratch workspace under `t.TempDir()`
3. Writes `_libtftest_override.tf.json` (provider endpoints) and
   `_libtftest_backend_override.tf.json` (forces local backend)
4. `tc.SetVar` sets the `bucket_name` variable with a unique prefix
5. `tc.Apply()` runs `terraform init` + `terraform apply`
6. `tc.Output` reads the `bucket_id` output
7. Assertions verify the bucket via AWS SDK calls to LocalStack
8. `t.Cleanup` runs: terraform destroy, then container stop

## Run It

```bash
go test -tags=integration -v -run TestS3Bucket ./test/...
```
