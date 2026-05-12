# Changelog

All notable changes to libtftest will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and libtftest adheres to [Semantic Versioning](https://semver.org/). While
the library is pre-1.0, minor-version bumps may contain breaking changes;
the API freeze begins at v1.0.

## [Unreleased]

## [0.2.0] - 2026-05-12

Tracks IMPL-0003. Adopts terratest v1.0's `*Context` paired-method API
across `TestCase`, `assert/`, and `fixtures/`. Non-context methods
remain as permanent shims that forward to the `*Context` variant with
`tb.Context()`. Cleanup paths use `context.WithoutCancel(tb.Context())`
so destroy + fixture teardown survive test-end cancellation.

### Added

- `TestCase.ApplyContext` / `ApplyContextE` / `PlanContext` /
  `PlanContextE` / `OutputContext` — caller-supplied context variants
- `assert/*` paired `*Context` methods for every helper:
  `BucketExistsContext`, `BucketHasEncryptionContext`,
  `BucketHasVersioningContext`, `BucketBlocksPublicAccessContext`,
  `BucketHasTagContext`, `TableExistsContext`, `RoleExistsContext`
  (Pro), `RoleHasInlinePolicyContext` (Pro), `ParameterExistsContext`,
  `ParameterHasValueContext`, `FunctionExistsContext`
- `fixtures/` paired `Seed*Context` functions for `SeedS3Object`,
  `SeedSSMParameter`, `SeedSecret`, `SeedSQSMessage`. Cleanup callbacks
  use `context.WithoutCancel(ctx)`
- `docs/examples/07-cancellation.md` walking through the ctx API
- `docs/examples/examples_integration_test.go` — runnable tests gated
  by the `integration_examples` build tag
- `make test-examples` target + CI step running the example tests

### Changed

- `terraform.InitAndApply`, `terraform.DestroyE`, etc. — all six call
  sites migrated from the deprecated non-context terratest helpers to
  their `*Context` variants
- Container startup (`New()`) and container teardown (`stack.Stop` in
  cleanup) now thread `tb.Context()` / `WithoutCancel(tb.Context())`
  instead of `context.Background()`
- Local skill templates (`libtftest:add-assertion`,
  `libtftest:add-fixture`) emit paired methods by default
- `.golangci.yml` — `context-as-argument` revive rule allows
  `testing.{T,B,F,TB}` before `context.Context`

### Removed

- Six `//nolint:staticcheck` SA1019 suppressions added by PR #8 (the
  terratest v1.0 bump). The deprecated `*` non-context terratest
  helpers are no longer called directly from libtftest

### Migration notes

The non-context API is unchanged — existing tests calling `tc.Apply()`,
`assert.S3.BucketExists`, `fixtures.SeedS3Object`, etc., continue to
work without source modification. New tests should prefer the `*Context`
variants when the test cares about deadlines, tracing, or external
cancellation; otherwise the shim form is equivalent.

## [0.1.0] - TBD

First public release. Tracked by IMPL-0001. Tag pending.
