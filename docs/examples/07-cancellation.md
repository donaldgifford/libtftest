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

Non-context methods (`tc.Apply`, `s3assert.BucketExists`, `s3fix.SeedObject`,
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
    s3assert "github.com/donaldgifford/libtftest/assert/s3"
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
    s3assert.BucketExistsContext(t, applyCtx, tc.AWS(), bucket)
}
```

## Cancellation propagates to downstream SDK calls

After a successful `Apply`, the AWS SDK clients you get from `tc.AWS()`
honor whatever context you hand them. Cancelling that context aborts
in-flight SDK calls cleanly:

```go
import (
    "github.com/aws/aws-sdk-go-v2/service/s3"
)

func TestS3Module_AssertionCancellation(t *testing.T) {
    t.Parallel()

    tc := libtftest.New(t, &libtftest.Options{
        Edition:   localstack.EditionCommunity,
        ModuleDir: "../../modules/s3-bucket",
    })
    tc.SetVar("bucket_name", tc.Prefix()+"-cancel-demo")
    tc.Apply()
    bucket := tc.Output("bucket_id")

    ctx, cancel := context.WithCancel(t.Context())
    cancel() // Cancellation propagates through the AWS SDK.

    client := s3.NewFromConfig(tc.AWS())
    _, err := client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: &bucket})
    if err == nil {
        t.Fatal("HeadBucket with cancelled ctx returned nil error")
    }
}
```

> **Note.** Don't pass a *pre-cancelled* context to `PlanContextE`
> or `ApplyContextE`. terratest v1.0's retry helper panics on a nil
> error description when the action returns before the retry loop can
> classify it. Per-call deadlines that fire mid-operation (above) work
> correctly; only pre-cancellation trips the upstream bug. For
> deterministic negative cancellation testing, exercise the AWS SDK
> via `tc.AWS()` as shown here.

## Cleanup paths use `context.WithoutCancel`

The `terraform destroy` callback that libtftest registers via `t.Cleanup`
uses `context.WithoutCancel(tb.Context())`. The same applies to
`fixtures/<service>.Seed*Context` cleanup callbacks. This means:

- Cancelling the test context does **not** cancel the destroy step
- Trace/value plumbing on `tb.Context()` is preserved through cleanup

```text
Test starts                tb.Context() = ctxA  (cancelled on test end)
  ctx := WithTimeout(ctxA)              = ctxB  (also cancelled on ctxA cancel)
  tc.ApplyContext(ctx)                  // honors ctxB cancellation
  s3fix.SeedObjectContext(t, ctx, ...)
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
