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
  - [Part 2 — TestCase.AssertIdempotent and TestCase.AssertIdempotentApply](#part-2--testcaseassertidempotent-and-testcaseassertidempotentapply)
    - [API](#api)
    - [Why on TestCase rather than assert/idempotency](#why-on-testcase-rather-than-assertidempotency)
  - [Part 3 — assert/tags package](#part-3--asserttags-package)
    - [API](#api-1)
    - [Why Resource Groups Tagging API rather than per-service ListTagsForResource](#why-resource-groups-tagging-api-rather-than-per-service-listtagsforresource)
  - [Part 4 — assert/snapshot package](#part-4--assertsnapshot-package)
    - [API](#api-2)
    - [Snapshot update protocol](#snapshot-update-protocol)
    - [Why not just cmp.Diff?](#why-not-just-cmpdiff)
    - [Extraction helpers](#extraction-helpers)
  - [Part 5 — Repo-wide doc.go convention](#part-5--repo-wide-docgo-convention)
    - [Rule](#rule)
    - [Deprecated top-level packages](#deprecated-top-level-packages)
    - [awsx/ flat-layout justification](#awsx-flat-layout-justification)
    - [Tooling](#tooling)
  - [Part 6 — tools/docgen marker scanner + feature matrix](#part-6--toolsdocgen-marker-scanner--feature-matrix)
    - [Marker grammar](#marker-grammar)
    - [Tool: tools/docgen](#tool-toolsdocgen)
    - [Make targets](#make-targets)
    - [Output location](#output-location)
    - [Why regex + go/parser rather than pure AST](#why-regex--goparser-rather-than-pure-ast)
- [API / Interface Changes](#api--interface-changes)
  - [Removed (renamed)](#removed-renamed)
  - [Added](#added)
  - [Conventions added (repo-wide)](#conventions-added-repo-wide)
- [Data Model](#data-model)
- [Testing Strategy](#testing-strategy)
- [Migration / Rollout Plan](#migration--rollout-plan)
  - [Sequencing](#sequencing)
  - [Backwards compatibility](#backwards-compatibility)
  - [Consumer migration](#consumer-migration)
  - [Skill template updates](#skill-template-updates)
- [Resolved Questions](#resolved-questions)
- [References](#references)
<!--toc:end-->

## Overview

Three orthogonal libtftest features fell out of [INV-0002][inv-0002]'s
EKS coverage analysis (Parts 2–4), plus one prerequisite refactor
(Part 1). Two follow-on conventions emerged during IMPL-0004
planning: a `doc.go`-per-package mandate (Part 5, from
[INV-0003][inv-0003]) and a marker-scanner / feature-matrix tool
(Part 6, from [INV-0004][inv-0004]). This design covers all six as
a single coordinated v0.2.0 release.

The story: when adding more services (EKS, ECS, SNS, SQS, KMS, …),
the current flat `assert/{service}.go` layout with zero-size-struct
namespacing (`assert.S3.BucketExists`) doesn't scale. We switch to
per-service sub-packages (`assert/s3/`, `fixtures/s3/`) that mirror
the AWS SDK v2 convention. While the layout is being touched, three
generic patterns from the EKS coverage matrix land as first-class
features, the package layout doubles as the rollout vehicle for a
repo-wide `doc.go` convention, and a small in-tree docgen tool
makes the "what needs Pro / mockta / X to run?" question
answerable from a single rendered markdown page.

[inv-0002]: ../investigation/0002-eks-coverage-via-localstack.md
[inv-0003]: ../investigation/0003-package-documentation-convention-and-gomarkdoc-toolchain.md
[inv-0004]: ../investigation/0004-pro-and-oss-feature-matrix-tooling.md

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
- Adopt a repo-wide `doc.go`-per-package convention so every Go
  package documents its intent in a canonical location
- Add a `// libtftest:requires <tag>[,<tag>...] <reason>` marker
  convention with a `tools/docgen` scanner that renders a
  `docs/feature-matrix.md` page and a `make check-markers` CI gate
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
- Wiring `gomarkdoc` (or any other godoc → markdown renderer) for
  the `doc.go` content. The convention itself ships here; the
  renderer ships in a follow-up DESIGN+IMPL
- Pushing the marker convention upstream into the `go-development`
  plugin. Lives in this repo until it's baked in for a release or
  two

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

### Part 2 — `TestCase.AssertIdempotent` and `TestCase.AssertIdempotentApply`

Two variants, both gated on the same notion of "module is idempotent":
the cheap one runs a fresh Plan after the caller's existing Apply; the
strict one performs the canonical **double-Apply** pattern
(Apply → Plan → Apply → Plan, asserting both plans are empty).

Catches different bug classes:

| Variant | Catches |
|---------|---------|
| `AssertIdempotent` (Plan-only) | Bad `ignore_changes`, provider refresh-time drift, `known-after-apply` placeholders that didn't resolve |
| `AssertIdempotentApply` (double-Apply) | Above + computed-vs-known mismatches that only surface on the second Apply, in-place updates the provider reports on Plan but reverts on Apply |

`AssertIdempotent` is the default — cheap, surfaces 80% of bugs.
`AssertIdempotentApply` is the rigorous variant for modules with
suspicious refresh behavior (KMS keys, IAM policies with `for_each`,
random resources with weird `triggers`).

#### API

```go
// AssertIdempotent runs Plan and fails the test if the plan reports
// any resource changes (add, change, or destroy). Use this once per
// test after the initial Apply has completed; it does NOT call Apply
// itself. Calls tb.Errorf on non-zero change count; the test
// continues running so additional assertions can surface their own
// failures.
func (tc *TestCase) AssertIdempotent() {
    tc.tb.Helper()
    tc.AssertIdempotentContext(tc.tb.Context())
}

func (tc *TestCase) AssertIdempotentContext(ctx context.Context) {
    tc.tb.Helper()
    result := tc.PlanContext(ctx)
    if result.Changes.Add+result.Changes.Change+result.Changes.Destroy > 0 {
        tc.tb.Errorf(
            "module is not idempotent: plan shows add=%d change=%d destroy=%d",
            result.Changes.Add, result.Changes.Change, result.Changes.Destroy,
        )
    }
}

// AssertIdempotentApply performs the canonical double-Apply check:
// runs Plan, fails if non-empty, then runs Apply again, then runs
// Plan again, failing if that's non-empty. Catches a strictly
// larger class of bugs than AssertIdempotent — including ones that
// only surface on the second Apply — at the cost of one extra
// terraform apply round-trip.
//
// Like AssertIdempotent, use this after the caller's initial Apply
// has completed.
func (tc *TestCase) AssertIdempotentApply() {
    tc.tb.Helper()
    tc.AssertIdempotentApplyContext(tc.tb.Context())
}

func (tc *TestCase) AssertIdempotentApplyContext(ctx context.Context) {
    tc.tb.Helper()
    // First plan — should be clean already.
    if result := tc.PlanContext(ctx); result.Changes.Add+
        result.Changes.Change+result.Changes.Destroy > 0 {
        tc.tb.Errorf("first plan not idempotent: add=%d change=%d destroy=%d",
            result.Changes.Add, result.Changes.Change, result.Changes.Destroy)
        return
    }
    // Second Apply — should be a no-op.
    tc.ApplyContext(ctx)
    // Final plan — should be clean.
    if result := tc.PlanContext(ctx); result.Changes.Add+
        result.Changes.Change+result.Changes.Destroy > 0 {
        tc.tb.Errorf("post-second-Apply plan not idempotent: add=%d change=%d destroy=%d",
            result.Changes.Add, result.Changes.Change, result.Changes.Destroy)
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

#### Extraction helpers

Producing the JSON to snapshot is the caller's job, but the common
case — pull IAM policy documents out of a `terraform show -json`
dump — gets a turnkey helper. Generic JSON path extraction is the
escape hatch for anything else.

```go
// ExtractIAMPolicies parses Terraform plan JSON (the output of
// `terraform show -json plan.out`) and returns one entry per
// aws_iam_role / aws_iam_policy / aws_iam_role_policy resource:
// the assume role policy, inline policies, and any managed
// policy attachments rendered as JSON documents. Keys are the
// resource address (e.g. "aws_iam_role.eks_node[0]") + a suffix
// distinguishing assume_role / inline:<name> / managed:<arn>.
//
// Use the returned bytes as the `actual` argument to
// JSONStructural to lock down IAM policy shapes against a
// golden file.
func ExtractIAMPolicies(planJSON []byte) (map[string][]byte, error)

// ExtractResourceAttribute returns the JSON bytes at the given
// path under planned_values.root_module.resources for a specific
// resource address. Use for non-IAM extraction when ExtractIAMPolicies
// doesn't cover the resource type.
//
// Example:
//   policy, _ := snapshot.ExtractResourceAttribute(
//       planJSON,
//       "aws_kms_key.main",
//       "policy",
//   )
//   snapshot.JSONStructural(t, policy, "testdata/snapshots/kms-policy.json")
func ExtractResourceAttribute(
    planJSON []byte,
    resourceAddress string,
    attributePath string,
) ([]byte, error)
```

`ExtractIAMPolicies` is the obvious EKS / IAM-heavy use case from
INV-0002. `ExtractResourceAttribute` is the general-purpose escape
hatch — covers KMS key policies, S3 bucket policies, etc., without
forcing libtftest to ship a helper per service.

### Part 5 — Repo-wide `doc.go` convention

Origin: [INV-0003][inv-0003], Concluded. The `go-development`
plugin currently mentions `doc.go` exactly once (one line in
`project-structure.md:226` as a suggestion in a file-layout
listing) — no spec, no enforcement, no rendering guidance. We
formalise it for this repo.

#### Rule

Every Go package in the repo (including `internal/...` and
`cmd/...`) ships a dedicated `doc.go`:

- Contains **only** the `package <name>` declaration and a
  godoc-compliant multi-paragraph package comment
- **No** imports, types, constants, or functions in `doc.go`
- The package comment opens with `// Package <name> ...` per
  godoc convention
- The existing `// Package <name>` block in whatever random `.go`
  file currently holds it (e.g. `assert/assert.go`,
  `awsx/config.go`, `tf/workspace.go`) moves into `doc.go` and is
  removed from the original file

#### Deprecated top-level packages

The Part 1 refactor leaves the top-level `assert/` and `fixtures/`
packages without any user-facing surface. Each gets a `doc.go`
that explicitly documents the deprecation and points readers to
the per-service sub-packages:

```go
// Package assert is deprecated. Use the per-service sub-packages
// instead:
//
//	import s3assert "github.com/donaldgifford/libtftest/assert/s3"
//	s3assert.BucketExists(t, cfg, name)
//
// See assert/s3, assert/dynamodb, assert/iam, assert/ssm,
// assert/lambda, assert/tags, assert/snapshot.
package assert
```

#### `awsx/` flat-layout justification

The Part 1 refactor explicitly does *not* break up `awsx/` into
sub-packages (see the original Resolved Question 1). The
`awsx/doc.go` file is where that intent gets documented — replaces
the otherwise-needed "deliberate non-change" CHANGELOG marker
that an earlier draft considered (and that triggered INV-0003 in
the first place).

#### Tooling

- **Renderer (`gomarkdoc` or equivalent) — deferred.** A future
  DESIGN+IMPL will wire a renderer behind `make docs` to emit the
  package surface to `docs/api/`. Out of scope here.
- **CI presence check — deferred.** A future tiny check
  (`scripts/check-doc-go.sh` or a Go program) will fail when a
  package directory lacks a `doc.go`. Out of scope here.

The *convention itself* is what ships in this design; tooling
follows once it's universal in the repo.

### Part 6 — `tools/docgen` marker scanner + feature matrix

Origin: [INV-0004][inv-0004], Concluded. Consumers can't discover
which assertions require LocalStack Pro (or future externals like
mockta for Okta) without reading source or hitting runtime skips.
Part 6 ships a marker convention plus a small in-tree Go tool
that renders a feature matrix and gates CI.

#### Marker grammar

```text
// libtftest:requires <tag>[,<tag>...] <free-text reason>
```

- Sits on its own line in the function's doc comment block
- Tags are a **comma-separated list with no whitespace inside**
  the list — keeps the regex parser simple and avoids ambiguity
  with the reason
- Reason is everything after the tag list. May contain commas,
  spaces, ASCII punctuation. Single line.
- Order within the tag list is not significant
- Tag set is open-ended — adding `mockta`, `lambda-docker`, etc.
  is just a marker-text change, no grammar rework

Example (single tag):

```go
// RoleHasInlinePolicy asserts that role has an inline policy named name.
//
// libtftest:requires pro IAM IdC managed policy ARNs only resolve under Pro
func RoleHasInlinePolicy(tb testing.TB, cfg aws.Config, role, name string) { ... }
```

Example (multi-tag):

```go
// OktaFederatedRoleHasTrust asserts the role trusts the Okta IdP.
//
// libtftest:requires pro,mockta EKS + Okta federation needs both
func OktaFederatedRoleHasTrust(tb testing.TB, ...) { ... }
```

#### Tool: `tools/docgen`

Single `package main` Go binary with three subcommands. Lives in
`tools/docgen/` and shares the libtftest go.mod — *not* a
separate module. **Intentionally does not import any libtftest
package**; scans source files with regex + `go/parser` for
declaration positions only. That keeps the tool version-agnostic
(no rebuild on library bumps) and avoids any import cycle.

| Subcommand | What it does |
|------------|--------------|
| `scan` | Walks `.go` files, pairs each marker line with the immediately following function declaration, emits a JSON IR (function name, package path, tags, reason, file:line). |
| `render` | Consumes the JSON IR, writes `docs/feature-matrix.md` — one row per marked function, one column per distinct tag, plus a "reason" column. |
| `check` | Walks for calls to `libtftest.RequirePro(` (regex + enclosing-function detection); fails non-zero with `file:line` when any such function lacks a marker. |

#### Make targets

```make
docs-matrix:
    go run ./tools/docgen render -o docs/feature-matrix.md

check-markers:
    go run ./tools/docgen check ./...

ci: lint test build license-check check-markers
```

`make check-markers` joins `make ci` so PRs catch missing markers.
`make docs-matrix` is invoked manually before tagged releases (or
by the release workflow after `Bump Version`); the regenerated
`docs/feature-matrix.md` lands in the release commit.

#### Output location

`docs/feature-matrix.md` at repo root level of `docs/`. Linked
from the top-level `README.md`. Not regenerated on every push to
avoid noisy diffs.

#### Why regex + `go/parser` rather than pure AST

The marker is a comment, not a Go construct. Regex scans the
comment text directly. `go/parser` is used only to find function
declaration positions so a marker can be unambiguously attached
to the function that immediately follows it. This split keeps the
scanner simple (~100 lines), fast (no full type-checking), and
robust against syntax errors in the surrounding code.

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
| `assert/snapshot` | `JSONStrict`, `JSONStructural`, `ExtractIAMPolicies`, `ExtractResourceAttribute` |
| `fixtures/s3` | `SeedObject`, `SeedObjectContext` |
| `fixtures/ssm` | `SeedParameter`, `SeedParameterContext` |
| `fixtures/secretsmanager` | `SeedSecret`, `SeedSecretContext` |
| `fixtures/sqs` | `SeedMessage`, `SeedMessageContext` |
| `libtftest.TestCase` | `AssertIdempotent`, `AssertIdempotentContext`, `AssertIdempotentApply`, `AssertIdempotentApplyContext` |
| `tools/docgen` | `package main` Go binary with `scan`, `render`, `check` subcommands |
| `Makefile` | `docs-matrix` + `check-markers` targets; `check-markers` wired into `ci` |
| `docs/feature-matrix.md` | Rendered output of `tools/docgen render` |
| `cliff.toml` | New `Tooling` group above `Features` for `feat(tools)` / `chore(tools)` commits |

`awsx/`, `localstack/`, `harness/`, `tf/`, `internal/`, and
`sneakystack/` have no semantic changes from this design, but each
gains a dedicated `doc.go` as part of the Part 5 rollout.

### Conventions added (repo-wide)

| Convention | Applies to | Spec |
|------------|-----------|------|
| `doc.go` per package | Every Go package, including `internal/...` and `cmd/...` | [Part 5](#part-5--repo-wide-docgo-convention) |
| `// libtftest:requires <tag>[,<tag>...] <reason>` marker | Every function that calls `libtftest.RequirePro(tb)` (or a future equivalent gate) | [Part 6](#part-6--toolsdocgen-marker-scanner--feature-matrix) |

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

**One feature branch, one PR, multiple commits, one release.**
Pre-1.0 SemVer gives us latitude to bundle the breaking layout
refactor with additive features; once we cross v1.0 we'll require
strict per-feature minor bumps.

The phase ordering within the branch matches IMPL-0004:

```text
Phase 1: refactor(assert)      — per-service package split
Phase 2: refactor(fixtures)    — per-service package split
Phase 3: docs                  — examples, README, CLAUDE.md, doc.go rollout
Phase 4: feat(libtftest)       — AssertIdempotent + AssertIdempotentApply
Phase 5: feat(assert/tags)     — RGT-backed tag propagation
Phase 6: feat(assert/snapshot) — JSON helpers + extraction
Phase 7: feat(tools/docgen)    — marker scanner + feature matrix + CI gate
Phase 8: (separate repo PR)    — claude-skills plugin v0.3.0
Phase 9: (cross-cutting)       — release verification after merge
```

Phase 1 must land before Phases 4–6 because the new feature
packages slot into the new layout. Phase 3 catches every
remaining call site and rolls out the `doc.go` convention to
existing packages. Phase 7 consumes the markers placed in
Phase 1 (`assert/iam`) and any others added during the
implementation.

### Backwards compatibility

- **None.** Pre-1.0 SemVer. The Phase 1–2 refactor is a breaking
  change for every consumer call site that uses
  `assert.<Service>` or `fixtures.Seed<Service><Resource>`. The
  CHANGELOG `[Changed]` entry spells out the find-and-replace
  pattern.
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

## Resolved Questions

1. **AWSX namespacing follows or stays flat?**
   **Resolved — stays flat.** `awsx/` is one ~10-line constructor per
   service in a single flat package, already idiomatic Go. If a future
   need arises to expose service-specific helpers beyond the
   constructor, revisit then. CHANGELOG entry calls this out as a
   deliberate non-change so future readers don't take it as an
   oversight.

2. **Should `assert/snapshot` ship a JSON-normalizer / extraction
   helper?**
   **Resolved — yes.** Ship `ExtractIAMPolicies(planJSON)` as the
   turnkey IAM-extraction helper (the obvious EKS / IAM-heavy use
   case) plus `ExtractResourceAttribute(planJSON, addr, path)` as a
   general-purpose escape hatch for non-IAM extraction. Spec'd in
   the [Part 4 Extraction helpers](#extraction-helpers) section.

3. **Does `AssertIdempotent` also re-run Apply or just Plan?**
   **Resolved — both.** Ship two variants:
   - `AssertIdempotent` (Plan only) — cheap default, surfaces 80% of
     bugs (bad `ignore_changes`, refresh-time drift, unresolved
     `known-after-apply`).
   - `AssertIdempotentApply` (double-Apply: Plan → Apply → Plan) —
     rigorous variant, catches the additional class of
     computed-vs-known mismatches that only surface on the second
     Apply.
   Spec'd in
   [Part 2](#part-2--testcaseassertidempotent-and-testcaseassertidempotentapply).

4. **Do we need a `_test.go` per `*Context` variant, or one per
   package?**
   **Resolved — one per package.** Carries forward the INV-0001
   pattern (one `<service>_test.go` per `<service>` package, with
   `fakeTB` + `*Context_PropagatesCancel` style tests inside it).

5. **Single PR + single minor bump, or split per part?**
   **Resolved — single PR, single `v0.2.0` minor bump.** Pre-1.0
   SemVer doesn't require strict per-feature minor splits; the
   breaking layout refactor already forces a minor bump and the
   three additive primitives (idempotency, tags, snapshot) plus
   the two conventions (doc.go, marker matrix) ride along. Once
   we cross v1.0 we'll revisit. See [IMPL-0004][impl-0004] for
   the phase-by-phase commit story.

6. **`fakeTB` location after the per-service split.**
   **Resolved — `internal/testfake/`.** The existing
   `assert/assert_test.go` stub becomes a shared package every
   per-service test file can import. Avoids duplicating the same
   fake `testing.TB` impl across `assert/s3/`, `assert/dynamodb/`,
   `assert/iam/`, etc.

7. **`ExtractIAMPolicies` return shape and managed-policy
   handling.**
   **Resolved — always deterministic, no network calls.** Inline
   policies extract as their full JSON document. AWS-managed and
   customer-managed policy attachments render as the canonical
   ARN string (treated as an enum-like identifier — AWS owns the
   managed ARNs, we don't, and fetching the live document at
   extraction time would make the helper network-dependent).
   Spec'd in [Part 4 Extraction helpers](#extraction-helpers).

8. **LocalStack OSS support for the Resource Groups Tagging API
   (Part 3 backend).**
   **Resolved — pick at implementation time.** Decision tree
   applied during Phase 5: if OSS supports it, ship as designed;
   if it partially supports it, mock the gap in `sneakystack/`
   (matches the IAM-IDC / Organizations pattern from DESIGN-0001);
   if it doesn't support it, gate `assert/tags` integration
   coverage behind `libtftest.RequirePro` and document the gate.
   General rule: for API-call gaps, prefer mock-in-sneakystack or
   `RequirePro` over standing up full alternatives.

9. **One combined example or three separate examples for the new
   primitives?**
   **Resolved — three separate examples.** Existing pattern
   (01-basic-s3 through 07-cancellation) is one concept per file,
   2–5 KB each. Bundling all three primitives into a single
   "module-hygiene" example would break the discoverability /
   linking pattern. Ship 08-idempotency.md, 09-tag-propagation.md,
   10-snapshot-iam.md.

[impl-0004]: ../impl/0004-module-hygiene-primitives-and-per-service-package-layout.md

## References

- [INV-0002 — EKS coverage via LocalStack][inv-0002] — origin of
  Parts 1–4 of this design
- [INV-0003 — Package documentation convention and gomarkdoc
  toolchain][inv-0003] — origin of Part 5
- [INV-0004 — Pro and OSS feature matrix tooling][inv-0004] —
  origin of Part 6
- [IMPL-0004 — Module hygiene primitives and per-service package
  layout][impl-0004] — implementation plan for all six parts
- [INV-0001 — terratest 1.0 context variant migration][inv-0001] —
  established the paired-method pattern this design preserves
- `aws-sdk-go-v2/service/<name>` — naming and layout precedent
- `testify/assert` and `testify/require` — flat per-package function
  precedent
- Terratest `modules/aws` — the flat-monolith pattern this moves
  away from

[inv-0001]: ../investigation/0001-terratest-10-context-variant-migration.md
