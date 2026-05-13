---
id: DESIGN-0003
title: "Module hygiene primitives and per-service package layout"
status: Draft
author: Donald Gifford
created: 2026-05-13
---
<!-- markdownlint-disable-file MD025 MD041 -->

# DESIGN 0003: Module hygiene primitives and per-service package layout

**Status:** Draft
**Author:** Donald Gifford
**Date:** 2026-05-13

<!--toc:start-->
- [Overview](#overview)
- [Goals and Non-Goals](#goals-and-non-goals)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Background](#background)
- [Detailed Design](#detailed-design)
  - [Part 1 — Package layout refactor (Option A2)](#part-1--package-layout-refactor-option-a2)
    - [Before](#before)
    - [After](#after)
    - [Function-name shape](#function-name-shape)
    - [Name collisions with AWS SDK](#name-collisions-with-aws-sdk)
  - [Part 2 — TestCase.AssertIdempotent](#part-2--testcaseassertidempotent)
    - [API](#api)
    - [Why on TestCase rather than assert/idempotency](#why-on-testcase-rather-than-assertidempotency)
  - [Part 3 — assert/tags package](#part-3--asserttags-package)
    - [API](#api-1)
    - [Why Resource Groups Tagging API rather than per-service ListTagsForResource](#why-resource-groups-tagging-api-rather-than-per-service-listtagsforresource)
  - [Part 4 — assert/snapshot package](#part-4--assertsnapshot-package)
    - [API](#api-2)
    - [Snapshot update protocol](#snapshot-update-protocol)
    - [Why not just cmp.Diff?](#why-not-just-cmpdiff)
- [API / Interface Changes](#api--interface-changes)
  - [Removed (renamed)](#removed-renamed)
  - [Added](#added)
- [Data Model](#data-model)
- [Testing Strategy](#testing-strategy)
- [Migration / Rollout Plan](#migration--rollout-plan)
  - [Sequencing](#sequencing)
  - [Backwards compatibility](#backwards-compatibility)
  - [Consumer migration](#consumer-migration)
  - [Skill template updates](#skill-template-updates)
- [Open Questions](#open-questions)
- [References](#references)
<!--toc:end-->

## Overview

Three orthogonal libtftest features fell out of [INV-0002][inv-0002]'s
EKS coverage analysis, plus one prerequisite refactor. This design
covers all four as a single coordinated design with independent
implementation PRs. The refactor (Part 1) lands first; the three
feature additions (Parts 2–4) land afterward in any order.

The story: when adding more services (EKS, ECS, SNS, SQS, KMS, …),
the current flat `assert/{service}.go` layout with zero-size-struct
namespacing (`assert.S3.BucketExists`) doesn't scale. We switch to
per-service sub-packages (`assert/s3/`, `fixtures/s3/`) that mirror
the AWS SDK v2 convention. While the layout is being touched, three
generic patterns from the EKS coverage matrix land as first-class
features so consumers don't reinvent them.

[inv-0002]: ../investigation/0002-eks-coverage-via-localstack.md

## Goals and Non-Goals

### Goals

- Replace `assert.<Service>.<Method>` (zero-size struct namespacing)
  with per-service Go packages: `assert/<service>` and `fixtures/<service>`
- Preserve the paired-method pattern (`Foo` + `FooContext`) on every
  function — orthogonal to layout
- Add `TestCase.AssertIdempotent(ctx)` for `apply` × 2 → empty plan
  assertion
- Add `assert/tags` package for root-tag propagation across resource
  ARNs (Resource Groups Tagging API backed)
- Add `assert/snapshot` package for JSON snapshot testing
  (LocalStack-independent; first consumer is IAM trust policy diffing)
- Update `tftest:add-assertion` / `tftest:add-fixture` plugin skills
  and the local `libtftest-add-assertion` / `libtftest-add-fixture`
  skills to emit the new shape

### Non-Goals

- Refactoring `awsx/` — already idiomatic; one file per service in a
  single flat package with `NewXxx(cfg)` is fine at any scale
- Interface-based design for `assert/*` or `fixtures/*` — no
  substitution surface, would be ceremony without payoff
- Shipping an `assert/eks` package — wait for a real consumer use case
  (will land naturally in the new layout when it does)
- Backwards-compat shim layer — pre-1.0, do this as a single coordinated
  PR with no re-exports
- Migrating consumer call sites in any consumer repo — that's a
  consumer-side find-and-replace; libtftest just ships the new shape
- Generating IAM snapshots automatically — `assert/snapshot` only
  compares; producing the JSON is the caller's responsibility
  (typically `terraform show -json | jq '.planned_values...'`)

## Background

INV-0002 surfaced three problems while sketching EKS coverage:

1. **Layout doesn't scale.** Current pattern is `assert/s3.go` with
   `var S3 = s3Asserts{}` and methods on a zero-size struct so callers
   write `assert.S3.BucketExists(...)`. Each new service is a new
   `var` and a new struct, accumulating in fewer-but-bigger files.
   At ~15 services this becomes unwieldy and godoc fragments badly.

2. **Three patterns kept showing up across hypothetical EKS tests
   that aren't EKS-specific:**
   - Apply-twice idempotency check
   - Tag propagation across resources
   - IAM policy snapshot diffing

   These three are module-hygiene primitives that every non-trivial
   Terraform module benefits from. Letting every consumer reinvent
   them is the wrong end of the cost curve.

3. **`assert.S3` doesn't compose well with imports.** Consumers
   importing both libtftest's `assert.S3` and the AWS SDK's `s3`
   package get a flat naming awkwardness — the libtftest version
   hides behind a struct method, the SDK version is a package-level
   constructor. Per-service packages line up better with the SDK
   convention consumers already know.

Prior art:

- `aws-sdk-go-v2/service/s3` — exact mirror of the proposed shape
- Terratest's `modules/aws` (the thing libtftest wraps) is the flat
  pattern we're moving away from — too late to fix there, but we
  don't have to inherit it
- `testify`'s `assert` and `require` packages — flat per-package
  function design; widely loved

## Detailed Design

### Part 1 — Package layout refactor (Option A2)

#### Before

```text
libtftest/
├── assert/
│   ├── assert.go          (package-level vars: var S3 = s3Asserts{})
│   ├── s3.go              (type s3Asserts struct{}; methods on it)
│   ├── dynamodb.go
│   ├── iam.go
│   ├── ssm.go
│   ├── lambda.go
│   └── assert_test.go
└── fixtures/
    ├── fixtures.go        (package-level functions)
    └── fixtures_test.go
```

#### After

```text
libtftest/
├── assert/
│   ├── s3/
│   │   ├── s3.go          (package s3: BucketExists, BucketExistsContext, ...)
│   │   └── s3_test.go
│   ├── dynamodb/
│   │   ├── dynamodb.go    (package dynamodb)
│   │   └── dynamodb_test.go
│   ├── iam/
│   │   ├── iam.go         (package iam)
│   │   └── iam_test.go
│   ├── ssm/
│   ├── lambda/
│   ├── tags/              (Part 3)
│   └── snapshot/          (Part 4)
└── fixtures/
    ├── s3/
    │   ├── s3.go          (package s3: SeedObject, SeedObjectContext, ...)
    │   └── s3_test.go
    ├── ssm/
    ├── secretsmanager/
    └── sqs/
```

Each service file becomes its own `package <service>`. Directory and
package name match (Go idiom). Tests co-located.

#### Function-name shape

Before:
```go
assert.S3.BucketExists(t, cfg, name)
assert.S3.BucketExistsContext(t, ctx, cfg, name)
fixtures.SeedS3Object(t, cfg, bucket, key, body)
fixtures.SeedS3ObjectContext(t, ctx, cfg, bucket, key, body)
```

After:
```go
import (
    s3assert "github.com/donaldgifford/libtftest/assert/s3"
    s3fix    "github.com/donaldgifford/libtftest/fixtures/s3"
)

s3assert.BucketExists(t, cfg, name)
s3assert.BucketExistsContext(t, ctx, cfg, name)
s3fix.SeedObject(t, cfg, bucket, key, body)
s3fix.SeedObjectContext(t, ctx, cfg, bucket, key, body)
```

Note the service prefix drops off the function name (`SeedS3Object`
→ `SeedObject`) because the package name already carries it.

#### Name collisions with AWS SDK

`package s3` in `assert/s3/` and `fixtures/s3/` will collide with
`github.com/aws/aws-sdk-go-v2/service/s3` at consumer call sites.
Resolution: import aliases (as shown above — `s3assert`, `s3fix`,
typically alongside `s3sdk` for the SDK). This is the same pattern
consumers already use when importing multiple `s3` packages and matches
testify's `assert` / `require` collision handling.

### Part 2 — `TestCase.AssertIdempotent`

Apply once, then call `terraform plan` again and assert zero
resource changes. Catches `ignore_changes` bugs, provider drift,
and computed-vs-known mismatches that only surface on the second
plan.

#### API

```go
// AssertIdempotent runs Apply followed by Plan and fails the test if
// the second plan reports any resource changes (add, change, or
// destroy). Use this once per test after the initial Apply has
// completed; it does NOT call Apply itself.
//
// Calls tb.Errorf on non-zero change count; the test continues running
// so additional assertions can surface their own failures.
func (tc *TestCase) AssertIdempotent() {
    tc.tb.Helper()
    tc.AssertIdempotentContext(tc.tb.Context())
}

// AssertIdempotentContext is the context-aware variant. The context
// is threaded into the inner PlanContext call.
func (tc *TestCase) AssertIdempotentContext(ctx context.Context) {
    tc.tb.Helper()
    result := tc.PlanContext(ctx)
    if result.Changes.Add+result.Changes.Change+result.Changes.Destroy > 0 {
        tc.tb.Errorf(
            "module is not idempotent: second plan shows add=%d change=%d destroy=%d",
            result.Changes.Add, result.Changes.Change, result.Changes.Destroy,
        )
    }
}
```

#### Why on `TestCase` rather than `assert/idempotency`

The check is a Terraform operation, not an AWS API call. It needs the
workspace, the vars, the backend config — all of which live on
`TestCase`. Pulling that out into a package-level function would
require a much wider API surface.

### Part 3 — `assert/tags` package

Verifies a baseline tag map is present on all listed resources via
the AWS Resource Groups Tagging API (`GetResources`). Service-agnostic
— anything with an ARN that the Tagging API can fetch.

#### API

```go
package tags

// PropagatesFromRoot asserts that every resource at the given ARNs
// carries every key/value pair in baseline. Extra tags on the
// resources are allowed (this is a subset check, not equality).
// Calls tb.Errorf on mismatch; collects errors across all resources
// rather than failing on the first.
func PropagatesFromRoot(
    tb testing.TB,
    cfg aws.Config,
    baseline map[string]string,
    arns ...string,
) {
    tb.Helper()
    PropagatesFromRootContext(tb, tb.Context(), cfg, baseline, arns...)
}

func PropagatesFromRootContext(
    tb testing.TB,
    ctx context.Context,
    cfg aws.Config,
    baseline map[string]string,
    arns ...string,
) {
    // resourcegroupstaggingapi.GetResources with ResourceARNList = arns
    // For each missing key or wrong value, tb.Errorf.
}
```

#### Why Resource Groups Tagging API rather than per-service `ListTagsForResource`

- One AWS call regardless of resource type
- Works for resources the per-service SDK packages don't yet have
  tag-listing helpers for in `awsx/`
- LocalStack Pro supports it; OSS coverage varies — falls back to
  per-service calls (a follow-up if needed)

### Part 4 — `assert/snapshot` package

Generic JSON snapshot testing. The caller supplies the JSON bytes;
the package compares against a snapshot file at a stable path and
either passes, fails with a diff, or — when `UPDATE_SNAPSHOTS=1` —
overwrites the file.

#### API

```go
package snapshot

// JSONStrict compares actual JSON bytes against the snapshot at path
// byte-for-byte. Use when key order matters or you want the strictest
// possible check. Fails on any difference; sets the failure message
// to a unified diff.
func JSONStrict(tb testing.TB, actual []byte, path string) {
    tb.Helper()
    // ... read snapshot, compare, fail with diff
}

// JSONStructural normalizes both actual and snapshot (sort keys,
// strip insignificant whitespace) before comparing. Use for IAM
// policies and any JSON where key order is not semantically
// meaningful.
func JSONStructural(tb testing.TB, actual []byte, path string) {
    tb.Helper()
    // ... normalize, compare, fail with diff
}
```

#### Snapshot update protocol

When `LIBTFTEST_UPDATE_SNAPSHOTS=1` is set, missing or mismatched
snapshots are overwritten with `actual`. This matches Jest's
`--updateSnapshot` flow and `go-cmp/cmp.Diff` workflows people
already use.

#### Why not just `cmp.Diff`?

- File I/O for the golden side is the value-add
- `JSONStructural` does normalization the caller would otherwise
  bake into every test
- Update-on-env-var is the killer feature

## API / Interface Changes

### Removed (renamed)

- `assert.S3`, `assert.DynamoDB`, `assert.IAM`, `assert.SSM`,
  `assert.Lambda` — zero-size struct vars and their methods
- `fixtures.SeedS3Object` and friends — flat package-level functions

### Added

| Path | Notes |
|------|-------|
| `assert/s3` | `BucketExists`, `BucketExistsContext`, `BucketHasEncryption`, `BucketHasVersioning`, `BucketBlocksPublicAccess`, `BucketHasTag` (+ `*Context` variants) |
| `assert/dynamodb` | `TableExists`, `TableExistsContext` |
| `assert/iam` | `RoleExists` (Pro), `RoleHasInlinePolicy` (Pro), + `*Context` variants |
| `assert/ssm` | `ParameterExists`, `ParameterHasValue`, + `*Context` variants |
| `assert/lambda` | `FunctionExists`, `FunctionExistsContext` |
| `assert/tags` | `PropagatesFromRoot`, `PropagatesFromRootContext` |
| `assert/snapshot` | `JSONStrict`, `JSONStructural` |
| `fixtures/s3` | `SeedObject`, `SeedObjectContext` |
| `fixtures/ssm` | `SeedParameter`, `SeedParameterContext` |
| `fixtures/secretsmanager` | `SeedSecret`, `SeedSecretContext` |
| `fixtures/sqs` | `SeedMessage`, `SeedMessageContext` |
| `libtftest.TestCase` | `AssertIdempotent`, `AssertIdempotentContext` |

`awsx/`, `localstack/`, `harness/`, `tf/`, `internal/`, and `sneakystack/`
are not touched by this design.

## Data Model

No persistent data model changes. The `assert/snapshot` package
introduces a convention for on-disk snapshot files but doesn't define
a strict schema — the caller's JSON is whatever the caller wrote.
Recommended location: `testdata/snapshots/<test>.json` per Go
convention.

## Testing Strategy

- **Unit:** each new `assert/{service}` and `fixtures/{service}` package
  carries its existing `_test.go` coverage forward. `*Context_PropagatesCancel`
  pattern from INV-0001 stays.
- **Unit:** `assert/tags` — fakeTB + stubbed Resource Groups Tagging
  API client (cancellation propagation + missing-tag scenarios).
- **Unit:** `assert/snapshot` — golden file diff scenarios:
  identical, byte-different but structurally equal (strict fails,
  structural passes), update mode rewrites file.
- **Integration:** add `TestCase.AssertIdempotent` to one existing
  integration test (e.g., `testdata/mod-s3` after the basic Apply).
  Confirms the API works end-to-end with real `terraform plan`.
- **Examples:** at least one example (`docs/examples/`) uses each
  of the four new APIs after the refactor lands.

## Migration / Rollout Plan

### Sequencing

```text
PR 1: Layout refactor (Part 1)
        ↓
PR 2: TestCase.AssertIdempotent (Part 2)
        ↓ ← independent of 3, 4
PR 3: assert/tags package (Part 3)
        ↓ ← independent of 2, 4
PR 4: assert/snapshot package (Part 4)
        ↓ ← independent of 2, 3
PR 5: claude-skills libtftest plugin bump (track 2 of issue #53)
```

PR 1 must land first because PRs 2–4 add packages in the new shape.
PRs 2–4 are orthogonal and can land in parallel after PR 1.

### Backwards compatibility

- **None.** Pre-1.0 SemVer. PR 1 is a breaking change for every
  consumer call site that uses `assert.<Service>` or
  `fixtures.Seed<Service><Resource>`. The CHANGELOG `[Changed]` entry
  spells out the find-and-replace pattern.
- No shim layer. Pre-1.0 the cost of carrying a deprecation tier is
  worse than the rip-the-bandaid PR cost.
- Version bump: **minor** (`v0.1.1` → `v0.2.0`). Pre-1.0 SemVer
  permits breaking changes on minor; the existing CHANGELOG header
  text covers this.

### Consumer migration

For each call site:

| Old | New |
|-----|-----|
| `assert.S3.BucketExists(t, cfg, b)` | `s3assert.BucketExists(t, cfg, b)` |
| `assert.S3.BucketExistsContext(t, ctx, cfg, b)` | `s3assert.BucketExistsContext(t, ctx, cfg, b)` |
| `fixtures.SeedS3Object(t, cfg, b, k, body)` | `s3fix.SeedObject(t, cfg, b, k, body)` |
| `fixtures.SeedS3ObjectContext(t, ctx, cfg, b, k, body)` | `s3fix.SeedObjectContext(t, ctx, cfg, b, k, body)` |

Plus an import shape change at the top of the file. A `sed` script
suffices for the function renames; the import block needs a manual
touch.

### Skill template updates

In `libtftest` repo (`.claude/skills/`):

- `libtftest-add-assertion` — template emits `package <service>` in
  `assert/<service>/<service>.go` instead of methods on a zero-size
  struct in `assert/<service>.go`
- `libtftest-add-fixture` — analogous for `fixtures/<service>/`

In `claude-skills` repo (`plugins/libtftest/skills/`):

- `tftest:add-assertion`, `tftest:add-fixture`, `tftest:add-test`,
  `tftest:scaffold` — all update example snippets and templates to
  use the new import + call shape
- Plugin version bump (independent SemVer): `0.2.0` → `0.3.0`
- Plugin pin range: `>=0.1.0, <1.0.0` → `>=0.2.0, <1.0.0` (the new
  libtftest minor)

## Open Questions

1. **AWSX namespacing follows or stays flat?** Today `awsx/s3.go`
   exports `NewS3(cfg)`. If we ever want to expose service-specific
   helpers beyond the constructor (e.g., `awsx.S3.PresignedURL`),
   we'd revisit. For now: stay flat. Note in the CHANGELOG so future
   readers know it was a conscious choice, not an oversight.

2. **Should `assert/snapshot` ship a JSON-normalizer helper?**
   Use case: caller does `terraform show -json`, wants the policy
   document, doesn't want to write the jq path. Probably yes as a
   later enhancement (`snapshot.ExtractIAMPolicies(planJSON)`); leave
   out of this design to keep the surface minimal.

3. **Does `AssertIdempotent` also re-run Apply or just Plan?**
   Currently designed as Plan-only (caller has already Applied). A
   re-Apply would catch a narrower class of bugs (refresh-time drift)
   at much higher cost. Plan-only is the right default; document the
   re-Apply path as "call `tc.Apply(); tc.AssertIdempotent()` then
   `tc.Apply()` again yourself" if a consumer needs it.

4. **Do we need a `_test.go` for each `_Context` variant in the new
   layout, or one file per package?** Tested in INV-0001 with one
   file per service top-level; same shape applies here with one
   file per per-service package. Carry forward.

## References

- [INV-0002 — EKS coverage via LocalStack][inv-0002] — origin of all
  four parts of this design
- [INV-0001 — terratest 1.0 context variant migration][inv-0001] —
  established the paired-method pattern this design preserves
- `aws-sdk-go-v2/service/<name>` — naming and layout precedent
- `testify/assert` and `testify/require` — flat per-package function
  precedent
- Terratest `modules/aws` — the flat-monolith pattern this moves away
  from

[inv-0001]: ../investigation/0001-terratest-10-context-variant-migration.md
