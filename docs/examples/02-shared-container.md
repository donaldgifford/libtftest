# Shared Container with TestMain

For faster test suites, share one LocalStack container across all tests in a
package using `harness.Run`. This avoids the 5-10 second container startup
overhead per test.

## Test Setup

```go
//go:build integration

package test

import (
    "testing"

    "github.com/donaldgifford/libtftest"
    ddbassert "github.com/donaldgifford/libtftest/assert/dynamodb"
    s3assert "github.com/donaldgifford/libtftest/assert/s3"
    "github.com/donaldgifford/libtftest/harness"
    "github.com/donaldgifford/libtftest/localstack"
)

// TestMain starts a shared LocalStack container for the entire package.
// All Test* functions in this package reuse it automatically.
func TestMain(m *testing.M) {
    harness.Run(m, harness.Config{
        Edition:  localstack.EditionCommunity,
        Services: []string{"s3", "dynamodb", "ssm"},
    })
}

func TestS3Bucket(t *testing.T) {
    t.Parallel()

    tc := libtftest.New(t, &libtftest.Options{
        ModuleDir: "../../modules/s3-bucket",
    })
    tc.SetVar("bucket_name", tc.Prefix()+"-s3-test")

    tc.Apply()

    s3assert.BucketExists(t, tc.AWS(), tc.Output("bucket_id"))
}

func TestDynamoDBTable(t *testing.T) {
    t.Parallel()

    tc := libtftest.New(t, &libtftest.Options{
        ModuleDir: "../../modules/dynamodb-table",
    })
    tc.SetVar("table_name", tc.Prefix()+"-dynamo-test")

    tc.Apply()

    ddbassert.TableExists(t, tc.AWS(), tc.Output("table_name"))
}
```

## How It Works

1. `harness.Run` starts one LocalStack container before any tests run
2. `libtftest.New` calls `harness.Current()` to detect the shared container
3. Each test gets its own scratch workspace but shares the container
4. `tc.Prefix()` ensures resource names don't collide across parallel tests
5. After all tests complete, `harness.Run` stops the container and exits

## When to Use

- **Per-package mode (shared):** most test suites -- faster, one container
- **Per-test mode (default):** max isolation, or when tests need different
  LocalStack configurations (different services, different editions)

## Prefix Collision Warning

If a test creates resources without using `tc.Prefix()`, the harness emits a
warning. Always embed the prefix in resource names:

```go
// Good
tc.SetVar("bucket_name", tc.Prefix()+"-my-bucket")

// Bad -- will collide with parallel tests
tc.SetVar("bucket_name", "my-bucket")
```
