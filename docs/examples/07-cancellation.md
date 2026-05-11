# Cancellation and Deadlines

> Runnable counterpart: `Test_Example07_Cancellation` in
> [`examples_integration_test.go`](examples_integration_test.go).

libtftest's `*Context` methods accept a caller-supplied `context.Context`,
enabling:

- **Per-call deadlines** — fail fast if a `terraform apply` exceeds a
  budget rather than letting it hang for the full test timeout
- **Cancellation coordination** — a parent goroutine cancelling causes
  the in-flight terraform/AWS-SDK calls to abort
- **Tracing propagation** — pass an OTel context through so terraform
  operations show up in distributed traces

Non-context methods (`tc.Apply`, `assert.S3.BucketExists`, `fixtures.SeedS3Object`,
etc.) are permanent shims that internally call the `*Context` variant
with `tb.Context()`. Use whichever fits the test.

## Per-call deadline

```go
//go:build integration

package test

import (
    "context"
    "testing"
    "time"

    "github.com/donaldgifford/libtftest"
    "github.com/donaldgifford/libtftest/assert"
    "github.com/donaldgifford/libtftest/localstack"
)

func TestS3Module_WithApplyDeadline(t *testing.T) {
    t.Parallel()

    tc := libtftest.New(t, &libtftest.Options{
        Edition:   localstack.EditionCommunity,
        ModuleDir: "../../modules/s3-bucket",
    })
    tc.SetVar("bucket_name", tc.Prefix()+"-deadline-test")

    // Bound the apply at 2 minutes regardless of the parent test timeout.
    applyCtx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
    defer cancel()

    tc.ApplyContext(applyCtx)

    // Assertions can use either tb.Context() (via the shim) or an
    // explicit context — both are fine.
    bucket := tc.OutputContext(applyCtx, "bucket_id")
    assert.S3.BucketExistsContext(t, applyCtx, tc.AWS(), bucket)
}
```

## Cancellation via cancelE()

```go
func TestS3Module_NegativePlanWithCancellation(t *testing.T) {
    t.Parallel()

    tc := libtftest.New(t, &libtftest.Options{
        Edition:   localstack.EditionCommunity,
        ModuleDir: "../../modules/s3-bucket",
    })

    ctx, cancel := context.WithCancel(t.Context())
    cancel() // Immediately cancel — PlanContextE should return an error.

    _, err := tc.PlanContextE(ctx)
    if err == nil {
        t.Fatal("PlanContextE with cancelled ctx returned nil error")
    }
}
```

## Cleanup paths use `context.WithoutCancel`

The `terraform destroy` callback that libtftest registers via `t.Cleanup`
uses `context.WithoutCancel(tb.Context())`. The same applies to
`fixtures.Seed*Context` cleanup callbacks. This means:

- Cancelling the test context does **not** cancel the destroy step
- Trace/value plumbing on `tb.Context()` is preserved through cleanup

```text
Test starts                tb.Context() = ctxA  (cancelled on test end)
  ctx := WithTimeout(ctxA)              = ctxB  (also cancelled on ctxA cancel)
  tc.ApplyContext(ctx)                  // honors ctxB cancellation
  fixtures.SeedS3ObjectContext(t, ctx, ...)
                                        // cleanup uses WithoutCancel(ctx)
                                        // -> survives test end
Test ends                  // ctxA, ctxB cancel
  destroy cleanup runs with WithoutCancel(ctxA)
  fixture cleanup runs with WithoutCancel(ctx)
```

## When to reach for the `*Context` variants

| Situation | Shim form | `*Context` form |
| --- | --- | --- |
| First test for a module | `tc.Apply()` | — |
| Custom deadline on a slow apply | — | `tc.ApplyContext(ctx)` |
| OTel tracing | — | `tc.ApplyContext(ctx)` |
| Coordinated cancellation with a sibling goroutine | — | `tc.ApplyContext(ctx)` |
| Negative test asserting cancellation | — | `tc.ApplyContextE(ctx)` |

When in doubt, start with the shim form. Migrate to `*Context` if the
test needs deadlines, tracing, or external cancellation signals.
