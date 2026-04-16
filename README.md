# libtftest

Terraform module integration testing library. Wraps [Terratest](https://terratest.gruntwork.io/) with opinionated, [LocalStack](https://localstack.cloud/)-aware defaults so module authors write ~10 lines of Go instead of ~200.

## Quick Start

```bash
go get github.com/donaldgifford/libtftest
```

```go
package test

import (
    "testing"

    "github.com/donaldgifford/libtftest"
    "github.com/donaldgifford/libtftest/assert"
    "github.com/donaldgifford/libtftest/localstack"
)

func TestS3Module(t *testing.T) {
    t.Parallel()

    tc := libtftest.New(t, &libtftest.Options{
        Edition:   localstack.EditionCommunity,
        ModuleDir: "../../modules/s3-bucket",
    })
    tc.SetVar("bucket_name", tc.Prefix()+"-logs")

    tc.Apply()

    bucket := tc.Output("bucket_id")
    assert.S3.BucketExists(t, tc.AWS(), bucket)
    assert.S3.BucketHasVersioning(t, tc.AWS(), bucket)
}
```

Everything else -- container startup, provider override, backend isolation, `terraform destroy`, container teardown -- is handled by `t.Cleanup` callbacks registered inside `libtftest.New`.

## Features

- **Zero-boilerplate container lifecycle** -- starts LocalStack, health checks, and cleans up automatically
- **Provider override injection** -- writes `_libtftest_override.tf.json` so your `.tf` files stay untouched
- **Backend isolation** -- forces `backend "local"` to prevent hitting real S3 backends
- **Parallel safety** -- `tc.Prefix()` generates unique resource name prefixes
- **Shared container mode** -- `harness.Run` in `TestMain` shares one container across a test package
- **Edition detection** -- `RequirePro(t)` auto-skips tests on Community edition
- **AWS SDK v2 clients** -- pre-configured `awsx` constructors for 11 services
- **Assertion helpers** -- `assert.S3`, `assert.IAM`, `assert.DynamoDB`, etc.
- **Fixture seeding** -- `fixtures.SeedS3Object`, `fixtures.SeedSSMParameter`, etc.
- **sneakystack** -- gap-filling proxy for LocalStack blind spots (IAM Identity Center, Organizations)

## Shared Container Mode

```go
func TestMain(m *testing.M) {
    harness.Run(m, harness.Config{
        Edition: localstack.EditionCommunity,
    })
}
```

## Documentation

- [Design Doc (DESIGN-0001)](docs/design/0001-libtftest-shared-terratest-localstack-harness-for-aws-modules.md)
- [Implementation Plan (IMPL-0001)](docs/impl/0001-libtftest-v010-core-library-implementation.md)
