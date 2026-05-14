# Idempotency Assertions

> Runnable counterpart: `Test_Example08_Idempotency` in
> [`examples_integration_test.go`](examples_integration_test.go).

Idempotency — running `terraform apply` twice on the same module
produces zero further changes — is the canonical health check for a
Terraform module. `libtftest` ships two assertion variants on `TestCase`:

| Method                                | Cost                     | Catches |
| ------------------------------------- | ------------------------ | --- |
| `tc.AssertIdempotent()`               | One extra `plan`         | Bad `ignore_changes`, refresh-time drift, unresolved `known-after-apply` placeholders |
| `tc.AssertIdempotentApply()`          | One extra `plan` + `apply` + `plan` | Above + computed-vs-known mismatches that only surface on the second `apply`, and in-place updates the provider reports on plan but reverts on apply |

`AssertIdempotent` is the cheap default — surfaces ~80% of bugs.
`AssertIdempotentApply` is the rigorous variant for modules with
suspicious refresh behavior (KMS keys, IAM policies driven by
`for_each`, `random_*` resources with weird `triggers`).

## Cheap variant — `AssertIdempotent`

```go
//go:build integration

package test

import (
    "testing"

    "github.com/donaldgifford/libtftest"
    s3assert "github.com/donaldgifford/libtftest/assert/s3"
    "github.com/donaldgifford/libtftest/localstack"
)

func TestS3Module_Idempotent(t *testing.T) {
    t.Parallel()

    tc := libtftest.New(t, &libtftest.Options{
        Edition:   localstack.EditionCommunity,
        ModuleDir: "../../modules/s3-bucket",
    })
    tc.SetVar("bucket_name", tc.Prefix()+"-idempotent")

    tc.Apply()

    bucket := tc.Output("bucket_id")
    s3assert.BucketExists(t, tc.AWS(), bucket)

    // One extra `plan` — fails the test if it reports any changes.
    tc.AssertIdempotent()
}
```

## Rigorous variant — `AssertIdempotentApply`

```go
func TestS3Module_DoubleApplyIdempotent(t *testing.T) {
    t.Parallel()

    tc := libtftest.New(t, &libtftest.Options{
        Edition:   localstack.EditionCommunity,
        ModuleDir: "../../modules/s3-bucket",
    })
    tc.SetVar("bucket_name", tc.Prefix()+"-double-apply")

    tc.Apply()

    // Plan → Apply → Plan. Both plans must be empty.
    tc.AssertIdempotentApply()
}
```

## With a deadline

The `*Context` variants accept a caller-supplied `context.Context`:

```go
import (
    "context"
    "time"
)

func TestS3Module_IdempotentWithDeadline(t *testing.T) {
    t.Parallel()

    tc := libtftest.New(t, &libtftest.Options{
        Edition:   localstack.EditionCommunity,
        ModuleDir: "../../modules/s3-bucket",
    })
    tc.SetVar("bucket_name", tc.Prefix()+"-deadline-idempotency")

    tc.Apply()

    ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
    defer cancel()

    tc.AssertIdempotentContext(ctx)
}
```

## When the check fails

`AssertIdempotent` calls `t.Errorf` (not `Fatalf`) on a non-zero change
count, so other assertions in the same test continue running. The
failure message reports the breakdown:

```
module is not idempotent: plan shows add=0 change=1 destroy=0
```

The follow-up is to inspect the plan JSON — `tc.Plan()` returns
`*PlanResult` whose `.JSON` field carries the raw plan output. Common
culprits:

- A provider-side default that drifts between plan and apply (e.g.
  the AWS provider rewriting an IAM policy's whitespace)
- An `ignore_changes` block that doesn't actually cover the field that
  drifts
- A `timestamp()` or `uuid()` call in module source — these are
  non-deterministic and produce a non-idempotent module by design

## When to reach for each

| Module shape                                   | Recommended |
| ---------------------------------------------- | --- |
| Simple resources (S3 buckets, DynamoDB tables) | `AssertIdempotent` |
| Modules with computed IAM policy documents     | `AssertIdempotentApply` |
| Modules with `random_*` resources              | `AssertIdempotentApply` |
| Modules with `for_each` over a computed map    | `AssertIdempotentApply` |
