# libtftest

Terraform module integration testing library. Wraps
[Terratest](https://terratest.gruntwork.io/) with opinionated,
[LocalStack](https://localstack.cloud/)-aware defaults so module authors write
~10 lines of Go instead of ~200.

## What it does

libtftest manages the full lifecycle of a Terraform module integration test:

1. Starts a LocalStack container (or reuses a shared one)
2. Copies your module to a scratch workspace
3. Injects provider and backend overrides so your `.tf` files stay untouched
4. Runs `terraform init` + `apply` (or `plan`) against LocalStack
5. Hands you pre-configured AWS SDK v2 clients for assertions
6. Cleans up everything via `t.Cleanup` -- destroy, stop container, flush logs

The module also includes **sneakystack**, a gap-filling HTTP proxy for
LocalStack blind spots (IAM Identity Center, Organizations, Control Tower). It
ships as both an importable Go package and a standalone Docker container.

## Install

```bash
go get github.com/donaldgifford/libtftest
```

**Requirements:** Go 1.25+, Docker (for running LocalStack containers),
Terraform CLI (installed via [mise](https://mise.jdx.dev/) or manually).

## Quick Start

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

Run it:

```bash
go test -tags=integration -v ./test/...
```

See [docs/examples](docs/examples/) for more complete examples.

## Features

- **Zero-boilerplate container lifecycle** -- `libtftest.New` starts LocalStack,
  health checks, and cleans up automatically
- **Provider override injection** -- writes `_libtftest_override.tf.json` so
  your `.tf` files stay untouched
- **Backend isolation** -- forces `backend "local"` via
  `_libtftest_backend_override.tf.json` to prevent hitting real S3 backends
- **Parallel safety** -- `tc.Prefix()` generates unique 10-char resource name
  prefixes (`ltt-` + 6 hex chars)
- **Shared container mode** -- `harness.Run` in `TestMain` shares one container
  across an entire test package
- **Edition detection** -- `RequirePro(t)` auto-skips tests on Community edition;
  Pro-only assertions call it internally
- **AWS SDK v2 clients** -- pre-configured `awsx` constructors for S3, DynamoDB,
  IAM, SSM, Secrets Manager, SQS, SNS, Lambda, KMS, Kinesis, STS
- **Assertion helpers** -- `assert.S3`, `assert.IAM`, `assert.DynamoDB`,
  `assert.SSM`, `assert.Lambda`
- **Fixture seeding** -- `fixtures.SeedS3Object`, `fixtures.SeedSSMParameter`,
  `fixtures.SeedSecret`, `fixtures.SeedSQSMessage` with automatic `t.Cleanup`
- **Plan testing** -- `tc.Plan()` returns parsed `PlanResult` with resource
  change counts for golden-file testing
- **sneakystack** -- gap-filling proxy for LocalStack blind spots, usable as an
  in-process sidecar or standalone Docker container
- **Configurable image** -- defaults to OSS LocalStack; override via
  `Options.Image` or `LIBTFTEST_LOCALSTACK_IMAGE` for Pro, airgapped, or custom
  images

## Shared Container Mode

For faster tests, share one LocalStack container across all tests in a package:

```go
package test

import (
    "testing"

    "github.com/donaldgifford/libtftest/harness"
    "github.com/donaldgifford/libtftest/localstack"
)

func TestMain(m *testing.M) {
    harness.Run(m, harness.Config{
        Edition: localstack.EditionCommunity,
    })
}

// Individual tests automatically reuse the shared container.
func TestMyModule(t *testing.T) {
    t.Parallel()
    tc := libtftest.New(t, &libtftest.Options{
        ModuleDir: "../",
    })
    // ...
}
```

## Package Overview

| Package | Purpose |
| --- | --- |
| `libtftest` | Core API: `TestCase`, `New`, `Apply`, `Plan`, `Output`, `SetVar` |
| `localstack` | Container lifecycle, edition detection, health polling |
| `tf` | Workspace copy, provider/backend override, terraform.Options builder |
| `awsx` | AWS SDK v2 client constructors configured for LocalStack |
| `fixtures` | Pre-apply data seeding with automatic cleanup |
| `assert` | Post-apply assertion helpers grouped by AWS service |
| `harness` | Shared-container `TestMain` helper, `Sidecar` interface |
| `sneakystack` | LocalStack gap-filling proxy with `Store` interface |

## Documentation

| Doc | Description |
| --- | --- |
| [Examples](docs/examples/) | Usage examples for common testing scenarios |
| [Development Guide](docs/development/) | How to develop, test, and contribute to libtftest |
| [Design Doc (DESIGN-0001)](docs/design/0001-libtftest-shared-terratest-localstack-harness-for-aws-modules.md) | Architecture and API design |
| [Implementation Plan (IMPL-0001)](docs/impl/0001-libtftest-v010-core-library-implementation.md) | Phased implementation plan |

## License

See [LICENSE](LICENSE) for details.
