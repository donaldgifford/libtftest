---
id: INV-0001
title: "Terratest 1.0 Context Variant Migration"
status: Concluded
author: Donald Gifford
created: 2026-05-11
---
<!-- markdownlint-disable-file MD025 MD041 -->

# INV 0001: Terratest 1.0 Context Variant Migration

**Status:** Concluded
**Author:** Donald Gifford
**Date:** 2026-05-11

<!--toc:start-->
- [Question](#question)
- [Hypothesis](#hypothesis)
- [Context](#context)
- [Approach](#approach)
- [Environment](#environment)
- [Findings](#findings)
  - [Observation 1: Terratest's 1.0 Context API shape](#observation-1-terratests-10-context-api-shape)
  - [Observation 2: libtftest's terraform.* call sites](#observation-2-libtftests-terraform-call-sites)
  - [Observation 3: libtftest already uses context.Background everywhere else](#observation-3-libtftest-already-uses-contextbackground-everywhere-else)
  - [Observation 4: testing.TB has Context() in Go 1.24+](#observation-4-testingtb-has-context-in-go-124)
  - [Observation 5: harness.Run builds its own timeout context](#observation-5-harnessrun-builds-its-own-timeout-context)
- [Design Options](#design-options)
  - [Option A: Internal-only — call tb.Context() at each site](#option-a-internal-only--call-tbcontext-at-each-site)
  - [Option B: Caller-supplied via Options.Context](#option-b-caller-supplied-via-optionscontext)
  - [Option C: New WithContext methods alongside existing](#option-c-new-withcontext-methods-alongside-existing)
  - [Option D: Store ctx on TestCase, default to tb.Context()](#option-d-store-ctx-on-testcase-default-to-tbcontext)
- [Resolved Questions](#resolved-questions)
- [Conclusion](#conclusion)
- [Recommendation](#recommendation)
- [References](#references)
<!--toc:end-->

## Question

How should libtftest adopt terratest 1.0's `*Context` variants for terraform
operations — what API shape minimizes churn for callers while giving us
real cancellation semantics, and should the migration extend beyond terratest
to the `assert/`, `fixtures/`, and cleanup paths that currently use
`context.Background()`?

## Hypothesis

Internalizing `tb.Context()` would be the smallest change, but it is also a
Go anti-pattern — it hides cancellation, deadlines, and tracing from the
caller. The right answer is to mirror terratest's own approach: keep
the existing single-signature methods (defaulting to `tb.Context()`)
and add `*Context` variants that accept a caller-supplied context. The
`assert/` and `fixtures/` sweeps should happen in the same pass so the
API stays consistent.

## Context

PR #8 bumped terratest to v1.0.0 with the minimum-effort path: six
`//nolint:staticcheck` annotations suppressing SA1019 deprecation warnings
on `InitAndApply`, `InitAndApplyE`, `InitAndPlanE`, `ShowE`, `Output`, and
`DestroyE`. The nolint comments point at this investigation.

**Triggered by:** PR #8 / terratest 1.0 release. The non-context variants
remain functional but deprecated; their removal in a future terratest
major would break libtftest if we do nothing.

## Approach

1. Inspect terratest 1.0's `modules/terraform/*.go` to confirm the
   `*Context` signature pattern and verify there are no behavioral
   differences between context and non-context variants.
2. Enumerate every libtftest call site that passes `context.Background()`
   (not just the terratest ones).
3. Compare four migration options against criteria: API churn, ergonomics,
   cancellation behavior, and consistency.
4. Identify any test-harness interactions that complicate the design
   (e.g., `harness.Run`'s pre-built timeout context).
5. Surface open questions for review before drafting an IMPL doc.

## Environment

| Component | Version / Value |
|-----------|----------------|
| terratest | v1.0.0 (pinned in go.mod) |
| Go (toolchain) | 1.26.1 |
| go.mod | `go 1.26` |
| testing.TB.Context() | available (Go 1.24+) |
| libtftest API | pre-1.0 (no compat guarantees yet) |

## Findings

### Observation 1: Terratest's 1.0 Context API shape

All six functions libtftest calls today have a matching `*Context` variant
with `ctx context.Context` inserted as the second parameter:

| Current (deprecated) | New canonical form |
|----------------------|--------------------|
| `terraform.InitAndApply(t, opts)` | `terraform.InitAndApplyContext(t, ctx, opts)` |
| `terraform.InitAndApplyE(t, opts)` | `terraform.InitAndApplyContextE(t, ctx, opts)` |
| `terraform.InitAndPlanE(t, opts)` | `terraform.InitAndPlanContextE(t, ctx, opts)` |
| `terraform.ShowE(t, opts)` | `terraform.ShowContextE(t, ctx, opts)` |
| `terraform.Output(t, opts, key)` | `terraform.OutputContext(t, ctx, opts, key)` |
| `terraform.DestroyE(t, opts)` | `terraform.DestroyContextE(t, ctx, opts)` |

The non-context variants now delegate to the context variants with
`context.Background()` internally, so behavior is identical at runtime
when ctx is `Background`. The context is used to cancel the underlying
`os/exec.CommandContext` invocation.

### Observation 2: libtftest's terraform.* call sites

Six total, all in `libtftest.go`:

- L146: `terraform.InitAndApply` (in `Apply`)
- L157: `terraform.InitAndApplyE` (in `ApplyE`)
- L181: `terraform.InitAndPlanE` (in `PlanE`)
- L187: `terraform.ShowE` (in `PlanE`)
- L211: `terraform.Output` (in `Output`)
- L283: `terraform.DestroyE` (in cleanup callback inside `registerCleanup`)

No terratest calls live outside `libtftest.go`. The `tf/` package only
builds `terraform.Options`, never executes.

### Observation 3: libtftest already uses context.Background everywhere else

Beyond the terratest sites, `context.Background()` appears at **15 call
sites** in non-test code, all AWS SDK or container ops:

- `libtftest.go`: 2 sites (`resolveContainer` startup, container Stop in cleanup)
- `assert/`: 10 sites (s3, dynamodb, iam, ssm, lambda)
- `fixtures/`: 4 sites (S3 put, DynamoDB put, SSM put, SQS send-message)
- `harness/testmain.go`: 2 sites (startup timeout + stopAll on failure)

The harness already builds a `context.WithTimeout(context.Background(), 3*time.Minute)`
for sidecar/container startup but **does not propagate it** to anything
the test itself uses.

### Observation 4: testing.TB has Context() in Go 1.24+

`testing.TB.Context() context.Context` was added in Go 1.24. It returns a
context that is canceled automatically when the test, subtest, or benchmark
ends — including on `t.Fatal`, panic, and `Cleanup` ordering. We're on
Go 1.26.1, so this is available.

This means `tb.Context()` is the natural "default" — using it gives free
cancellation when the test ends, with no plumbing required.

### Observation 5: harness.Run builds its own timeout context

`harness/testmain.go:53` builds `ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)`
for the shared-container startup path. This is independent of any test's
`tb.Context()`. If a per-test operation should inherit cancellation from
the harness's context, that's a new requirement; today it doesn't.

The simpler read: the harness context governs **container/sidecar
lifecycle**, while `tb.Context()` would govern **per-test terraform
operations**. Keeping them separate is fine.

## Design Options

### Option A: Internal-only — call tb.Context() at each site

```go
func (tc *TestCase) Apply() *terraform.Options {
    tc.tb.Helper()
    tfOpts := tf.BuildOptions(tc.tb, tc.work.Dir, tc.vars)
    terraform.InitAndApplyContext(tc.tb, tc.tb.Context(), tfOpts)
    return tfOpts
}
```

- **Pros:** zero public API change. Mechanical, low risk. Cancellation
  comes "for free" when the test ends.
- **Cons:** callers can't pass deadlines or use custom contexts (e.g.,
  one with tracing). Loses one of the main reasons terratest added the
  context plumbing in the first place.

### Option B: Caller-supplied via Options.Context

```go
type Options struct {
    // ... existing fields
    Context context.Context // optional; defaults to tb.Context()
}
```

Stored on `TestCase`, used at every site.

- **Pros:** callers gain a way to plumb deadlines or shared
  cancellation. Zero new methods.
- **Cons:** a `context.Context` on `Options` is awkward — typical Go
  style is to pass ctx as a method argument, not a config field. Also
  static for the lifetime of the TestCase, which limits per-call
  deadline use.

### Option C: New WithContext methods alongside existing

```go
func (tc *TestCase) Apply() *terraform.Options             { return tc.ApplyContext(tc.tb.Context()) }
func (tc *TestCase) ApplyContext(ctx context.Context) *terraform.Options { ... }
```

- **Pros:** mirrors terratest 1.0's own pattern. Existing callers
  unaffected; new callers opt in. Per-call deadlines work.
- **Cons:** API surface doubles. Six new methods (eight if you count
  the E variants).

### Option D: Store ctx on TestCase, default to tb.Context()

```go
type TestCase struct {
    // ...
    ctx context.Context
}

// In New(): tc.ctx = tb.Context()

func (tc *TestCase) WithContext(ctx context.Context) *TestCase {
    tc.ctx = ctx
    return tc
}
```

- **Pros:** clean ergonomics — `New(t, opts).WithContext(ctx).Apply()`.
  Methods stay single-signature.
- **Cons:** mutable state on TestCase, which the current design avoids
  (vars map being the only exception). Builder pattern feels heavier
  than the rest of the API.

## Resolved Questions

1. **Should the migration extend to `assert/` and `fixtures/`?**
   **Yes — full sweep.** All 14 `context.Background()` call sites
   in those packages migrate to caller-supplied ctx with the same
   pattern. Users see one consistent API, one breakage, one migration
   note. Examples in `docs/examples/` will be updated as part of the
   IMPL.

2. **Do we want caller-supplied contexts at all?**
   **Yes — required.** Hiding ctx inside the library is a Go
   anti-pattern: it blocks per-call deadlines, OpenTelemetry tracing
   propagation, and parent-goroutine cancellation coordination. Even
   if no current consumer asks for it, the surface should be there.
   This decision rules out Options A, B, and D.

3. **Should the harness's 3-minute startup context become the parent
   of `tb.Context()`?**
   **No — non-issue.** The harness context governs only `harness.Run`
   internals (sidecar startup). By the time tests execute,
   `harness.Run` has already returned and that context is done.
   They are correctly independent.

4. **The cleanup `Destroy` call runs after the test ends — does
   `tb.Context()` still work there?**
   **No — use `context.WithoutCancel(tb.Context())` for cleanup.**
   `tb.Context()` is canceled at test end, so the destroy callback
   would fire with a canceled ctx. `context.WithoutCancel` (Go 1.21+)
   preserves trace/value plumbing without inheriting the cancellation,
   which is exactly what cleanup needs.

5. **INV numbering mismatch in PR #8's nolint comments.**
   **Resolved.** PR #8's `INV-0004` references will be fixed to
   `INV-0001` on this branch as part of the doc PR (one mechanical
   `sed`-style edit).

## Conclusion

The migration is straightforward: **Option C** (paired `*Context`
methods alongside existing single-signature methods) is the canonical
Go pattern and what terratest itself did for the same reason. Scope is
the full sweep across terratest, `assert/`, `fixtures/`, and the
`context.Background()` sites in `libtftest.go`. Cleanup paths use
`context.WithoutCancel(tb.Context())`.

**Answer:** Yes — adopt Option C, full-sweep scope, `context.WithoutCancel`
for cleanup. Proceed to IMPL.

## Recommendation

Open `IMPL-0003: Terratest 1.0 context migration` with these phases:

1. **API surface for `libtftest.TestCase`** — add `ApplyContext`,
   `ApplyContextE`, `PlanContext`, `PlanContextE`, `OutputContext`,
   plus rename the cleanup-internal destroy to use
   `context.WithoutCancel(tc.tb.Context())`. Keep existing methods
   as thin shims that call the new variants with `tc.tb.Context()`.
2. **`assert/` package** — every method gains a `ctx context.Context`
   parameter. Update the 10 call sites in `s3.go`, `dynamodb.go`,
   `iam.go`, `ssm.go`, `lambda.go`. Same shim pattern: existing
   non-ctx methods delegate to ctx variants with `tb.Context()`.
3. **`fixtures/` package** — same treatment for the 4 Seed* helpers.
4. **`libtftest.go` non-test cleanups** — replace the two remaining
   `context.Background()` sites (`resolveContainer`, `stack.Stop`)
   with the appropriate `tb.Context()` / `WithoutCancel` variants.
5. **Remove SA1019 nolints** — delete the six `//nolint:staticcheck`
   annotations once the call sites use `*Context` variants.
6. **Update examples and skills** — `docs/examples/`, local skill
   templates (`libtftest:add-assertion`, `libtftest:add-fixture`),
   and the consumer plugin templates in `donaldgifford/claude-skills`.
7. **CHANGELOG** — call out the API change. Pre-1.0, so this is a
   minor-version bump with breaking notes, not a major.

Suggested acceptance criteria: zero `staticcheck SA1019` remaining,
zero `context.Background()` outside of `_test.go`, examples compile
and pass, all CI green.

## References

- [PR #8: chore(deps): bump terratest to v1.0.0](https://github.com/donaldgifford/libtftest/pull/8)
- [Terratest 1.0 release blog](https://www.gruntwork.io/blog/terratest-1-0-released)
- Terratest source: `~/go/pkg/mod/github.com/gruntwork-io/terratest@v1.0.0/modules/terraform/`
- `testing.TB.Context()` — Go 1.24 release notes
- `context.WithoutCancel` — Go 1.21 release notes
