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

**Requirements:** Go 1.26+ (uses `testing.TB.Context()`, requires 1.24
minimum), Docker (for running LocalStack containers), Terraform CLI
(installed via [mise](https://mise.jdx.dev/) or manually).

## Quick Start

```go
package test

import (
    "testing"

    "github.com/donaldgifford/libtftest"
    s3assert "github.com/donaldgifford/libtftest/assert/s3"
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
    s3assert.BucketExists(t, tc.AWS(), bucket)
    s3assert.BucketHasVersioning(t, tc.AWS(), bucket)
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
- **Assertion helpers** -- per-service packages under `assert/`:
  `s3assert`, `ddbassert`, `iamassert` (Pro), `ssmassert`, `lambdaassert`
  (importable as aliases to coexist with the AWS SDK)
- **Fixture seeding** -- per-service packages under `fixtures/`:
  `s3fix.SeedObject`, `ssmfix.SeedParameter`, `secretsfix.SeedSecret`,
  `sqsfix.SeedMessage` with automatic `t.Cleanup`
- **Plan testing** -- `tc.Plan()` returns parsed `PlanResult` with resource
  change counts for golden-file testing
- **Idempotency assertions** -- `tc.AssertIdempotent()` runs a fresh Plan and
  fails the test if any changes are pending;
  `tc.AssertIdempotentApply()` performs the rigorous double-Apply check
  (Plan -> Apply -> Plan, both plans empty). Both ship `*Context` variants
  for per-call deadlines. See
  [docs/examples/08-idempotency.md](docs/examples/08-idempotency.md).
- **Tag propagation assertion** -- `tagsassert.PropagatesFromRoot` calls the
  AWS Resource Groups Tagging API once and verifies a baseline tag map is
  present on every listed ARN. Subset check (extra tags allowed), aggregated
  failure messages, paired `*Context` variant. See
  [docs/examples/09-tag-propagation.md](docs/examples/09-tag-propagation.md).
- **JSON snapshot testing** -- `snapshot.JSONStrict` and
  `snapshot.JSONStructural` lock down deterministic JSON payloads against
  a golden file. `LIBTFTEST_UPDATE_SNAPSHOTS=1` regenerates snapshots in
  place. `snapshot.ExtractIAMPolicies` and
  `snapshot.ExtractResourceAttribute` pull policy documents out of
  `terraform show -json plan.out` output. See
  [docs/examples/10-snapshot-iam.md](docs/examples/10-snapshot-iam.md).
- **terratest 1.0 `*Context` API** -- every `TestCase` method, every
  per-service assertion, and every per-service fixture ships a paired `*Context`
  variant (`ApplyContext`, `BucketExistsContext`, `SeedObjectContext`,
  etc.); non-context forms are permanent shims that forward to the
  `*Context` variant with `tb.Context()`. Cleanup paths use
  `context.WithoutCancel` so destroy + teardown survive test-end
  cancellation. See [docs/examples/07-cancellation.md](docs/examples/07-cancellation.md).
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
| [Cancellation & ctx](docs/examples/07-cancellation.md) | `*Context` paired API, deadlines, `WithoutCancel` cleanup |
| [Idempotency](docs/examples/08-idempotency.md) | `tc.AssertIdempotent` and `tc.AssertIdempotentApply` |
| [CHANGELOG](CHANGELOG.md) | Released versions and migration notes |
| [Feature Matrix](docs/feature-matrix.md) | Generated table of Pro / mockta / multi-tag gated functions |
| [Development Guide](docs/development/) | How to develop, test, and contribute to libtftest |
| [Design Doc (DESIGN-0001)](docs/design/0001-libtftest-shared-terratest-localstack-harness-for-aws-modules.md) | Architecture and API design |
| [Implementation Plan (IMPL-0001)](docs/impl/0001-libtftest-v010-core-library-implementation.md) | Phased implementation plan |
| [Skills Design (DESIGN-0002)](docs/design/0002-claude-skills-for-libtftest-authors-and-consumers.md) | Claude Code skills for authors and consumers |
| [Skills Implementation (IMPL-0002)](docs/impl/0002-claude-skills-for-libtftest-authors-and-consumers.md) | Phased plan for the skills above |
| [Investigation (INV-0001)](docs/investigation/0001-terratest-10-context-variant-migration.md) | terratest 1.0 `*Context` migration analysis |
| [Context Migration (IMPL-0003)](docs/impl/0003-terratest-10-context-migration.md) | Phased plan that produced the paired-method API |

## Using Claude Code with libtftest

Two sets of Claude Code skills accelerate libtftest workflows:

**For libtftest maintainers** — local skills in this repo's `.claude/skills/`
that scaffold the most common new-code paths (assertions, fixtures,
sneakystack handlers, AWS clients) and a `libtftest-reviewer` agent that
catches libtftest-specific mistakes (PortEndpoint vs Endpoint, RequirePro
gating, `tb` naming). They activate automatically when working in this repo.

**For Terraform module repos that consume libtftest** — install the
`libtftest` plugin from
[`donaldgifford/claude-skills`](https://github.com/donaldgifford/claude-skills/tree/main/plugins/libtftest):

```bash
claude plugin install donaldgifford/claude-skills:libtftest
```

The plugin provides `tftest:scaffold` (bootstrap a `test/` directory),
`tftest:setup-ci` (wire the reusable GHA workflow), `tftest:add-test`,
`tftest:add-fixture`, `tftest:add-assertion`, `tftest:debug`,
`tftest:enable-pro`, `tftest:enable-sneakystack`, `tftest:upgrade`, plus a
`tftest-reviewer` agent.

See [docs/examples/README.md](docs/examples/README.md) for the full skill
list with descriptions.

## License

See [LICENSE](LICENSE) for details.
