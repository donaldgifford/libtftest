---
id: IMPL-0004
title: "Module hygiene primitives and per-service package layout"
status: Draft
author: Donald Gifford
created: 2026-05-13
---
<!-- markdownlint-disable-file MD025 MD041 -->

# IMPL 0004: Module hygiene primitives and per-service package layout

**Status:** Draft
**Author:** Donald Gifford
**Date:** 2026-05-13

<!--toc:start-->
- [Objective](#objective)
- [Scope](#scope)
  - [In Scope](#in-scope)
  - [Out of Scope](#out-of-scope)
- [Versioning Strategy](#versioning-strategy)
- [PR-to-Phase Mapping](#pr-to-phase-mapping)
- [Implementation Phases](#implementation-phases)
  - [Phase 1: `assert/` per-service refactor](#phase-1-assert-per-service-refactor)
  - [Phase 2: `fixtures/` per-service refactor](#phase-2-fixtures-per-service-refactor)
  - [Phase 3: Cross-cutting layout work](#phase-3-cross-cutting-layout-work)
  - [Phase 4: `TestCase.AssertIdempotent` + `AssertIdempotentApply`](#phase-4-testcaseassertidempotent--assertidempotentapply)
  - [Phase 5: `assert/tags` package](#phase-5-asserttags-package)
  - [Phase 6: `assert/snapshot` package](#phase-6-assertsnapshot-package)
  - [Phase 7: claude-skills plugin sync](#phase-7-claude-skills-plugin-sync)
  - [Phase 8: Release verification](#phase-8-release-verification)
- [File Changes](#file-changes)
- [Testing Plan](#testing-plan)
- [Dependencies](#dependencies)
- [Open Questions](#open-questions)
- [References](#references)
<!--toc:end-->

## Objective

Execute the four-PR plan defined in DESIGN-0003: a per-service package
layout refactor followed by three module-hygiene primitives
(`AssertIdempotent` + double-Apply variant, `assert/tags`,
`assert/snapshot`), plus the matching `claude-skills` plugin update.

**Implements:** [DESIGN-0003][design-0003]
(originated from [INV-0002][inv-0002])

[design-0003]: ../design/0003-module-hygiene-primitives-and-per-service-package-layout.md
[inv-0002]: ../investigation/0002-eks-coverage-via-localstack.md

## Scope

### In Scope

- Per-service sub-packages under `assert/` and `fixtures/`
- `package`-level functions replacing zero-size-struct namespacing
- `TestCase.AssertIdempotent` / `AssertIdempotentContext` /
  `AssertIdempotentApply` / `AssertIdempotentApplyContext`
- New `assert/tags` package: `PropagatesFromRoot` /
  `PropagatesFromRootContext`
- New `assert/snapshot` package: `JSONStrict`, `JSONStructural`,
  `ExtractIAMPolicies`, `ExtractResourceAttribute`
- New `internal/testfake` package for shared `fakeTB` reuse across
  per-service test files
- All `docs/examples/*.md` and the runnable
  `examples_integration_test.go` updated to the new import shape
- Local Claude Code skills (`.claude/skills/libtftest-add-assertion`,
  `libtftest-add-fixture`) updated to emit the new shape
- claude-skills plugin (`tftest:add-test`, `tftest:add-assertion`,
  `tftest:add-fixture`, `tftest:scaffold`) updated to emit the new
  shape; plugin version bump 0.2.0 → 0.3.0; pin range bump
- libtftest version bump per [Versioning Strategy](#versioning-strategy)

### Out of Scope

- Refactoring `awsx/` — stays flat per DESIGN-0003 rationale
- Backwards-compatibility shim layer or re-exports
- Migrating consumer call sites in any consumer repo
- Auto-generating IAM snapshots — caller produces JSON via
  `terraform show -json`; `assert/snapshot` only compares
- An `assert/eks` package — wait for a real consumer use case
- The `infrastructure-as-code` plugin's generic terratest skill
  update (tracked separately under claude-skills issue #53 Track 2)

## Versioning Strategy

DESIGN-0003 specifies a minor bump for the breaking layout change
(`v0.1.1` → `v0.2.0`). The remaining three PRs ship additive features.
Under this project's pre-1.0 SemVer (header in `CHANGELOG.md`), the
two valid interpretations are:

- **(a) Strict SemVer:** every new public surface is `minor` →
  `v0.2.0` (Part 1), `v0.3.0` (Part 2), `v0.4.0` (Part 3), `v0.5.0`
  (Part 4). Four releases.
- **(b) "Minor for breaking, patch for additive" (pre-1.0
  pragmatism):** `v0.2.0` (Part 1, breaking layout) → `v0.2.1`
  (Part 2), `v0.2.2` (Part 3), `v0.2.3` (Part 4).

**Recommendation: (a) Strict SemVer.** Each new feature is a real
addition to the public API surface and rewards consumers with a
clean per-version changelog story. The existing labeler workflow
already supports per-PR `minor` labels. See [open question
1](#open-questions).

## PR-to-Phase Mapping

| PR | Phases | Bump | Target tag |
|----|--------|------|------------|
| **PR 1** — Layout refactor | 1, 2, 3 | minor (breaking) | v0.2.0 |
| **PR 2** — `AssertIdempotent` + double-Apply | 4 | minor (additive) | v0.3.0 |
| **PR 3** — `assert/tags` | 5 | minor (additive) | v0.4.0 |
| **PR 4** — `assert/snapshot` | 6 | minor (additive) | v0.5.0 |
| **PR 5** — claude-skills plugin sync | 7 | n/a (plugin SemVer) | plugin 0.3.0 |

Phase 8 (release verification) is cross-cutting and applies to every
PR's CI.

## Implementation Phases

Each phase builds on the previous. A phase is complete when all its
tasks are checked off and its success criteria are met.

---

### Phase 1: `assert/` per-service refactor

Move each `assert/<service>.go` to its own per-service package. Drop
the zero-size-struct namespacing pattern in favor of package-level
functions. Preserve the paired `Foo` / `FooContext` shape from
INV-0001.

The shared `fakeTB` stub currently lives in `assert/assert_test.go`
and is referenced by every per-service `*Context_PropagatesCancel`
test. Move it to a new `internal/testfake` package so each
per-service test file can import it without duplication.

#### Tasks

- [ ] Create `internal/testfake/testfake.go` with the existing
      `fakeTB` minimal `testing.TB` surface (Helper, Errorf, Error,
      Fatalf, Fatal, Skip, Skipf, SkipNow, Logf, Log, Context)
- [ ] Create `assert/s3/s3.go` (`package s3`) with `BucketExists`,
      `BucketExistsContext`, `BucketHasEncryption`,
      `BucketHasEncryptionContext`, `BucketHasVersioning`,
      `BucketHasVersioningContext`, `BucketBlocksPublicAccess`,
      `BucketBlocksPublicAccessContext`, `BucketHasTag`,
      `BucketHasTagContext` as package-level functions
- [ ] Create `assert/s3/s3_test.go` with the existing S3 test coverage
      using `internal/testfake.NewFakeTB(...)`
- [ ] Create `assert/dynamodb/dynamodb.go` (`package dynamodb`) with
      `TableExists`, `TableExistsContext`
- [ ] Create `assert/dynamodb/dynamodb_test.go`
- [ ] Create `assert/iam/iam.go` (`package iam`) with `RoleExists`,
      `RoleExistsContext`, `RoleHasInlinePolicy`,
      `RoleHasInlinePolicyContext` (preserve `libtftest.RequirePro(tb)`
      gates)
- [ ] Create `assert/iam/iam_test.go`
- [ ] Create `assert/ssm/ssm.go` (`package ssm`) with
      `ParameterExists`, `ParameterExistsContext`, `ParameterHasValue`,
      `ParameterHasValueContext`
- [ ] Create `assert/ssm/ssm_test.go`
- [ ] Create `assert/lambda/lambda.go` (`package lambda`) with
      `FunctionExists`, `FunctionExistsContext`
- [ ] Create `assert/lambda/lambda_test.go`
- [ ] Delete `assert/s3.go`, `assert/dynamodb.go`, `assert/iam.go`,
      `assert/ssm.go`, `assert/lambda.go`
- [ ] Delete `assert/assert.go` if it only held the zero-size struct
      vars (`var S3 = s3Asserts{}` etc.); keep otherwise
- [ ] Delete `assert/assert_test.go` once `fakeTB` is migrated and
      every per-service file has its coverage

#### Success Criteria

- `go build ./...` succeeds
- `go test ./assert/...` passes
- No file remains at `assert/<service>.go` (top-level)
- `grep -rn "var S3 = s3Asserts" .` returns zero hits
- `grep -rn "assert.S3.BucketExists" .` returns zero hits in source
  (matches in markdown docs are addressed in Phase 3)

---

### Phase 2: `fixtures/` per-service refactor

Move the single `fixtures/fixtures.go` into per-service packages with
short function names (the package name carries the service prefix).

#### Tasks

- [ ] Create `fixtures/s3/s3.go` (`package s3`) with `SeedObject`,
      `SeedObjectContext`
- [ ] Create `fixtures/s3/s3_test.go` carrying forward the existing
      S3 fixture cancellation + cleanup-registered tests
- [ ] Create `fixtures/ssm/ssm.go` (`package ssm`) with
      `SeedParameter`, `SeedParameterContext`
- [ ] Create `fixtures/ssm/ssm_test.go`
- [ ] Create `fixtures/secretsmanager/secretsmanager.go`
      (`package secretsmanager`) with `SeedSecret`, `SeedSecretContext`
- [ ] Create `fixtures/secretsmanager/secretsmanager_test.go`
- [ ] Create `fixtures/sqs/sqs.go` (`package sqs`) with `SeedMessage`,
      `SeedMessageContext`
- [ ] Create `fixtures/sqs/sqs_test.go`
- [ ] Each per-service test imports `internal/testfake`
- [ ] Delete `fixtures/fixtures.go` and `fixtures/fixtures_test.go`
- [ ] Verify `context.WithoutCancel(ctx)` cleanup pattern survives the
      move

#### Success Criteria

- `go build ./...` succeeds
- `go test ./fixtures/...` passes
- No file remains at `fixtures/fixtures.go`
- `grep -rn "fixtures.SeedS3Object" .` returns zero hits in source

---

### Phase 3: Cross-cutting layout work

Update everything that referenced the old layout: docs, examples,
local skill templates, internal callers. After Phase 3, PR 1 is
ready to land.

#### Tasks

- [ ] `grep -rn 'assert\.\(S3\|DynamoDB\|IAM\|SSM\|Lambda\)\.' .` —
      enumerate every remaining call site (likely only in docs +
      examples after Phases 1–2)
- [ ] `grep -rn 'fixtures\.Seed' .` — enumerate every remaining
      seed-call call site
- [ ] Update `docs/examples/01-basic-s3-test.md` to use new import
      shape (`s3assert`, `s3fix`)
- [ ] Update `docs/examples/03-plan-testing.md`
- [ ] Update `docs/examples/04-fixtures.md`
- [ ] Update `docs/examples/07-cancellation.md`
- [ ] Update `docs/examples/README.md` if it has API surface examples
- [ ] Update `docs/examples/examples_integration_test.go` —
      regenerate runnable tests against the new layout; verify they
      still compile under `//go:build integration_examples`
- [ ] Update `README.md` "Features", "Quick Start", "Package
      Overview", and any other API-surface sections
- [ ] Update `CLAUDE.md` status line + Context API surface section
- [ ] Update `.claude/skills/libtftest-add-assertion/SKILL.md` to
      describe the new shape
- [ ] Update `.claude/skills/libtftest-add-assertion/references/assertion-template.go.tmpl`
      — emit `package <service>` + package-level functions instead of
      zero-size struct + methods
- [ ] Update `.claude/skills/libtftest-add-fixture/SKILL.md`
- [ ] Update `.claude/skills/libtftest-add-fixture/references/fixture-template.go.tmpl`
- [ ] Run `claudelint run .claude/` clean (or verify the CI
      `skills.yml` job stays green if claudelint is not in the local
      toolchain)
- [ ] Run `make fmt` and `make lint` clean
- [ ] Run `make ci` clean (lint + test + build + license-check)

#### Success Criteria

- All `docs/examples/` markdown uses the new import shape
- `go test -tags=integration_examples -v ./docs/examples/...` passes
  locally (or in CI if Docker isn't available)
- Local skill templates emit per-service-package code by default
- `make ci` clean
- PR 1 ready to open with the `minor` label

---

### Phase 4: `TestCase.AssertIdempotent` + `AssertIdempotentApply`

Implement both variants of the idempotency check. Lives in
`libtftest.go` next to the other `TestCase` methods (Apply, Plan,
Output) per DESIGN-0003's "lives on TestCase" rationale.

#### Tasks

- [ ] Add `AssertIdempotent()` shim and `AssertIdempotentContext(ctx)`
      to `libtftest.go`
- [ ] Add `AssertIdempotentApply()` shim and
      `AssertIdempotentApplyContext(ctx)` to `libtftest.go`
- [ ] Doc comments must end with periods (godot linter); each shim
      must have a `// <Name> is a shim that calls <Name>Context with
      tb.Context().` line
- [ ] Add a unit test confirming both variants are wired (compile-time
      method-signature check, in the libtftest_test.go style)
- [ ] Add `TestAssertIdempotent_S3Module` integration test in
      `libtftest_integration_test.go` — happy path (idempotent S3
      module passes both variants)
- [ ] Add `TestAssertIdempotent_DetectsDrift` integration test —
      injects synthetic drift via a `local-exec` provisioner that
      changes a non-managed resource between Apply and the
      idempotency check, asserts the check fails
- [ ] Update `docs/examples/` with a new `08-idempotency.md` example
      + matching `Test_Example08_Idempotency` in
      `examples_integration_test.go`
- [ ] Update `docs/examples/README.md` index
- [ ] Update `README.md` Features list to mention idempotency
      assertions
- [ ] Update `CLAUDE.md` Context API surface section
- [ ] Run `make ci` clean

#### Success Criteria

- Both variants compile and have doc comments
- Integration tests pass against LocalStack (`make test-integration`)
- Synthetic drift test fails the assertion as expected
- PR 2 ready to open with the `minor` label

---

### Phase 5: `assert/tags` package

Service-agnostic tag propagation assertion backed by the AWS Resource
Groups Tagging API (`resourcegroupstaggingapi.GetResources`).

#### Tasks

- [ ] Add `awsx/resourcegroupstaggingapi.go` with
      `NewResourceGroupsTagging(cfg aws.Config)` constructor
- [ ] Create `assert/tags/tags.go` (`package tags`) with
      `PropagatesFromRoot(tb, cfg, baseline, arns...)` and
      `PropagatesFromRootContext(tb, ctx, cfg, baseline, arns...)`
- [ ] Implement subset-check semantics: every key/value in `baseline`
      must be present on every ARN; extra tags on the resource are
      allowed
- [ ] Collect errors across all ARNs before calling `tb.Errorf` —
      surface "resource X is missing tag Y" + "resource X has tag Y
      with value Z, expected W" all at once
- [ ] Verify LocalStack OSS support for the Resource Groups Tagging
      API — if it's incomplete, add a Pro gate or fall back to
      per-service `ListTagsForResource` (see [open question
      4](#open-questions))
- [ ] Create `assert/tags/tags_test.go` with unit tests via
      `internal/testfake` covering: missing key, wrong value,
      multiple-ARN aggregation, cancellation propagation
- [ ] Add `assert/tags` integration test in
      `libtftest_integration_test.go` (or new package-local file)
      using a small Terraform module that creates 2–3 tagged
      resources
- [ ] Update `docs/examples/` with a new `09-tag-propagation.md`
      example + matching runnable test
- [ ] Update `docs/examples/README.md` index
- [ ] Update `README.md` Features list
- [ ] Run `make ci` clean

#### Success Criteria

- `assert/tags` package compiles and tests pass
- Integration test against LocalStack passes (OSS or Pro per Q4
  resolution)
- PR 3 ready to open with the `minor` label

---

### Phase 6: `assert/snapshot` package

Generic JSON snapshot testing with an `UPDATE_SNAPSHOTS=1` rewrite
protocol, plus IAM-specific and general-purpose extraction helpers
for Terraform plan JSON.

#### Tasks

- [ ] Create `assert/snapshot/snapshot.go` (`package snapshot`) with
      `JSONStrict(tb, actual, path)` and
      `JSONStructural(tb, actual, path)`
- [ ] Implement structural normalization: recursively sort keys,
      strip insignificant whitespace, normalize numeric types where
      JSON's spec is ambiguous
- [ ] Wire `LIBTFTEST_UPDATE_SNAPSHOTS=1` rewrite protocol — on
      mismatch, overwrite `path` with `actual` and pass the test;
      log via `tb.Logf` so CI runs surface what was overwritten
- [ ] Implement `ExtractIAMPolicies(planJSON []byte) (map[string][]byte, error)`
      — walks `planned_values.root_module.resources` for
      `aws_iam_role`, `aws_iam_policy`, `aws_iam_role_policy`; returns
      one entry per role per policy keyed by
      `<resource_address>.<assume_role|inline:<name>|managed:<arn>>`
- [ ] Implement `ExtractResourceAttribute(planJSON, addr, path) ([]byte, error)`
      — generic JSON path extraction under
      `planned_values.root_module.resources[?address==addr].values.<path>`
- [ ] Create `assert/snapshot/snapshot_test.go` covering: identical
      JSON, byte-different-but-structurally-equal (strict fails,
      structural passes), missing snapshot file (without update mode
      → fail; with update mode → write + pass), structurally
      different JSON (both forms fail)
- [ ] Create `assert/snapshot/extract_test.go` covering: extract IAM
      policies from a fixture plan JSON, extract a KMS key policy
      via the generic helper, missing resource address (returns
      error)
- [ ] Generate fixture plan JSON for tests: small Terraform module
      with one IAM role + one KMS key, capture
      `terraform show -json plan.out` as `testdata/plan-iam-kms.json`
- [ ] Update `docs/examples/` with a new `10-snapshot-iam.md`
      example + matching runnable test
- [ ] Update `docs/examples/README.md` index
- [ ] Update `README.md` Features list
- [ ] Run `make ci` clean

#### Success Criteria

- `assert/snapshot` package compiles and tests pass (no LocalStack
  required)
- Extraction helpers correctly parse the fixture plan JSON
- `LIBTFTEST_UPDATE_SNAPSHOTS=1` mode works locally
- PR 4 ready to open with the `minor` label

---

### Phase 7: claude-skills plugin sync

Bump the consumer-facing plugin to track the new libtftest layout
and feature set. Mirrors the work done for v0.1.0 in
`feat/libtftest-plugin-v0.2.0`.

#### Tasks

- [ ] Bump `plugins/libtftest/.claude-plugin/plugin.json` version
      0.2.0 → 0.3.0
- [ ] Matching bump in `.claude-plugin/marketplace.json`
- [ ] Version-pin range across all `tftest:*` skill bodies,
      `_version-check.md`, `_frontmatter.md`, `README.md`, and the
      reviewer agent: `>=0.1.0, <1.0.0` → `>=0.2.0, <1.0.0`
- [ ] Update `tftest:add-test` SKILL.md + scaffold to use the new
      import shape
- [ ] Update `tftest:add-assertion` SKILL.md + scaffold to use the
      new per-service-package shape
- [ ] Update `tftest:add-fixture` SKILL.md + scaffold to use the new
      per-service-package shape
- [ ] Update `tftest:scaffold` (single-layout template) to use the
      new import shape; add `AssertIdempotent` mention as a
      module-hygiene convention
- [ ] Update umbrella `tftest` SKILL.md to surface the new
      module-hygiene primitives (idempotency, tags, snapshot)
- [ ] Update `plugins/libtftest/CHANGELOG.md` with a `[0.3.0]` entry
      explaining: the libtftest v0.5.0 (or whichever final tag) API
      changes the plugin tracks, what changed for skill consumers,
      and the SemVer split
- [ ] Update `plugins/libtftest/README.md` API surface tables
- [ ] Run `make test-plugin PLUGIN=libtftest` clean
- [ ] Run `scripts/sync_readme.py` clean
- [ ] Run `git-cliff -o CHANGELOG.md` and commit the result as
      `chore(changelog): ...`

#### Success Criteria

- Plugin manifest at version 0.3.0
- Pin range points at libtftest minor that ships these features
- All `tftest:*` skills emit per-service-package code by default
- Plugin tests + sync-readme clean
- PR 5 ready to open in the `claude-skills` repo

---

### Phase 8: Release verification

Cross-cutting verification that applies to every PR's CI and the
final tagged releases. Not a separate PR; happens within each of
PRs 1–5.

#### Tasks

- [ ] PR 1 CI green (lint, test, integration, docker, drift check,
      claudelint)
- [ ] PR 1 merges, `Bump Version` + `Release` + `Changelog Sync` +
      `Docker` jobs all green on the main-branch run; v0.2.0 tag +
      GH Release published; multi-arch `sneakystack` image at
      `ghcr.io/donaldgifford/sneakystack:0.2.0` signed
- [ ] PR 2 CI green; v0.3.0 published the same way
- [ ] PR 3 CI green; v0.4.0 published the same way
- [ ] PR 4 CI green; v0.5.0 published the same way
- [ ] PR 5 (claude-skills) CI green; plugin v0.3.0 merges; the
      plugin's pin range now matches a real libtftest tag
- [ ] No `chore(deps)` dependabot PRs left orphaned
- [ ] `CHANGELOG.md` on main reflects v0.2.0–v0.5.0 sections
      produced by `git-cliff` without any manual fixups beyond the
      `chore(awsx)` no-change marker (see [open question 3](#open-questions))
- [ ] Update memory `MEMORY.md` pointer to a new memory entry
      summarizing the layout-change shape (deferred to post-merge of
      PR 4)

#### Success Criteria

- Five tags exist: libtftest v0.2.0, v0.3.0, v0.4.0, v0.5.0; plugin
  0.3.0
- Every tagged libtftest release has an associated GitHub Release
  with goreleaser-rendered notes
- Every tagged release has a signed multi-arch
  `ghcr.io/donaldgifford/sneakystack:<tag>` image
- IMPL-0004 doc status flipped to `Completed`
- DESIGN-0003 doc status flipped to `Accepted`
- INV-0002 doc status flipped to `Concluded` (it's already implied,
  but explicit is better)

---

## File Changes

### Created

| File | Purpose |
|------|---------|
| `internal/testfake/testfake.go` | Shared `fakeTB` for per-service test packages |
| `assert/s3/s3.go` + `_test.go` | Migrated S3 assertions |
| `assert/dynamodb/dynamodb.go` + `_test.go` | Migrated DynamoDB assertions |
| `assert/iam/iam.go` + `_test.go` | Migrated IAM assertions |
| `assert/ssm/ssm.go` + `_test.go` | Migrated SSM assertions |
| `assert/lambda/lambda.go` + `_test.go` | Migrated Lambda assertions |
| `assert/tags/tags.go` + `_test.go` | `PropagatesFromRoot` + ctx variant |
| `assert/snapshot/snapshot.go` + `_test.go` | JSON snapshot diffing |
| `assert/snapshot/extract.go` + `_test.go` | IAM + generic plan-JSON extraction |
| `assert/snapshot/testdata/plan-iam-kms.json` | Fixture plan JSON |
| `awsx/resourcegroupstaggingapi.go` | New constructor for tags backend |
| `fixtures/s3/s3.go` + `_test.go` | Migrated S3 fixtures |
| `fixtures/ssm/ssm.go` + `_test.go` | Migrated SSM fixtures |
| `fixtures/secretsmanager/secretsmanager.go` + `_test.go` | Migrated Secrets Manager fixtures |
| `fixtures/sqs/sqs.go` + `_test.go` | Migrated SQS fixtures |
| `docs/examples/08-idempotency.md` | Idempotency example |
| `docs/examples/09-tag-propagation.md` | Tag propagation example |
| `docs/examples/10-snapshot-iam.md` | Snapshot IAM example |

### Modified

| File | Reason |
|------|--------|
| `libtftest.go` | Add `AssertIdempotent` family (4 methods) |
| `libtftest_integration_test.go` | Add idempotency + drift detection tests |
| `docs/examples/01-basic-s3-test.md` | New import shape |
| `docs/examples/03-plan-testing.md` | New import shape |
| `docs/examples/04-fixtures.md` | New import shape |
| `docs/examples/07-cancellation.md` | New import shape |
| `docs/examples/README.md` | Index Phase 4–6 examples |
| `docs/examples/examples_integration_test.go` | Add tests for examples 8–10; update existing for new shape |
| `README.md` | Features list + Quick Start + Package Overview |
| `CLAUDE.md` | Status line + Context API surface |
| `CHANGELOG.md` | Regenerated by git-cliff after each phase |
| `.claude/skills/libtftest-add-assertion/SKILL.md` + template | Emit per-service-package shape |
| `.claude/skills/libtftest-add-fixture/SKILL.md` + template | Emit per-service-package shape |
| (claude-skills) `plugins/libtftest/.claude-plugin/plugin.json` | Version 0.3.0 |
| (claude-skills) `.claude-plugin/marketplace.json` | Version 0.3.0 |
| (claude-skills) `plugins/libtftest/skills/*/SKILL.md` | Pin range + new import shape |
| (claude-skills) `plugins/libtftest/skills/tftest-scaffold/**` | Template emits new shape |
| (claude-skills) `plugins/libtftest/CHANGELOG.md` | `[0.3.0]` entry |
| (claude-skills) `plugins/libtftest/README.md` | API surface tables |

### Deleted

| File | Reason |
|------|--------|
| `assert/s3.go`, `dynamodb.go`, `iam.go`, `ssm.go`, `lambda.go` | Migrated to per-service packages |
| `assert/assert.go` | Zero-size struct vars are gone (if file held only those) |
| `assert/assert_test.go` | `fakeTB` migrated to `internal/testfake`; per-service tests own coverage |
| `fixtures/fixtures.go` | Migrated to per-service packages |
| `fixtures/fixtures_test.go` | Per-service tests own coverage |

## Testing Plan

- [ ] Unit test: each new `assert/<service>` package mirrors the
      coverage the pre-refactor file had — at minimum the
      `*Context_PropagatesCancel` test per assertion
- [ ] Unit test: each new `fixtures/<service>` package covers the
      cancellation + `WithoutCancel` cleanup pattern from INV-0001
- [ ] Unit test: `assert/tags` covers missing-key, wrong-value,
      multi-ARN aggregation, ctx propagation
- [ ] Unit test: `assert/snapshot` covers identical / structurally-
      equal / different / missing-file / update-mode scenarios for
      both strict and structural variants
- [ ] Unit test: `assert/snapshot.ExtractIAMPolicies` against fixture
      plan JSON containing `aws_iam_role` + `aws_iam_policy`
- [ ] Unit test: `assert/snapshot.ExtractResourceAttribute` against
      a non-IAM resource type (KMS key)
- [ ] Integration test: end-to-end `TestCase.AssertIdempotent` against
      a known-idempotent S3 module
- [ ] Integration test: end-to-end `TestCase.AssertIdempotentApply`
      same module — succeeds despite the extra Apply round-trip
- [ ] Integration test: synthetic drift causes `AssertIdempotent` to
      fail
- [ ] Integration test: `assert/tags.PropagatesFromRoot` against a
      module that tags 2–3 resources with a known baseline
- [ ] `make ci` green on every PR
- [ ] `make test-coverage` shows no coverage regression
- [ ] `make test-examples` (Docker required) — every new
      `examples/0N-*.md` has a green matching test
- [ ] CHANGELOG drift check green on every PR

## Dependencies

- libtftest v0.1.1 (the current latest tag) as the baseline — every
  PR rebases on a stable main
- terratest v1.0.x — no version bump needed; this work doesn't
  change terratest usage
- Go 1.26 — already in `go.mod`
- LocalStack OSS `2026.04.0` (CI default) and Pro
  `2026.5.0.dev121` locally — see INV-0002. The
  `assert/tags` Resource Groups Tagging API coverage is the only
  EKS-adjacent concern; see [open question 4](#open-questions)
- AWS SDK v2 `resourcegroupstaggingapi` client (new direct dep for
  `awsx/resourcegroupstaggingapi.go`)
- claude-skills repo PR landed (Phase 7) before the libtftest
  `tftest:*` skills can be advertised as compatible

## Open Questions

1. **Versioning strategy across PRs 1–4.** Per
   [Versioning Strategy](#versioning-strategy):
   - **(a) Four minor bumps:** v0.2.0 (layout), v0.3.0 (idempotency),
     v0.4.0 (tags), v0.5.0 (snapshot). Each PR ships a clean
     per-version changelog.
   - **(b) One minor + three patches:** v0.2.0 (layout) → v0.2.1
     (idempotency) → v0.2.2 (tags) → v0.2.3 (snapshot). Treats
     additive features as patches under pre-1.0 "minor for
     breaking, patch for additive" pragmatism.
   - Recommendation: **(a)**. Per-feature minor bumps stay clean
     even past v1.0 when this convention will outlast pre-1.0
     SemVer's looseness.

2. **`fakeTB` location.** Move to `internal/testfake/` so each
   per-service test package can import it without duplication. Any
   reason to prefer per-package duplication instead?

3. **`awsx/` "deliberate non-change" CHANGELOG marker.** DESIGN-0003
   says note in CHANGELOG that `awsx/` stays flat on purpose. Since
   git-cliff drives CHANGELOG from commits, the only way to surface
   this in the v0.2.0 section is a `chore(awsx): keep flat package
   layout intentionally` empty commit. Worth doing? Or just rely on
   DESIGN-0003 + INV-0002 for the historical record?

4. **LocalStack support for Resource Groups Tagging API.** Phase 5
   depends on it. If OSS coverage is incomplete:
   - **(a)** Gate `assert/tags` integration tests behind `RequirePro`
   - **(b)** Implement a fallback path that falls back to per-service
     `ListTagsForResource` calls
   - **(c)** Ship `assert/tags` as unit-only and let consumers
     provide their own integration coverage
   - Need to verify against `localstack/localstack:2026.04.0` before
     committing.

5. **`assert/snapshot.ExtractIAMPolicies` return shape.** Current
   spec returns `map[string][]byte` keyed by
   `<resource_address>.<assume_role|inline:<name>|managed:<arn>>`.
   Concerns:
   - Map ordering is non-deterministic in Go — but the *caller*
     iterates and calls `JSONStructural` per entry, so ordering
     doesn't affect the snapshot file. Probably fine.
   - Managed policies live on the role only as ARN refs, not
     embedded JSON. Either we don't include them (rename to
     "PoliciesFromRole" with managed-arn references) or we fetch
     the managed policy document via the AWS SDK at extraction time
     (turns this into a "needs `aws.Config`" function). Pick one.

6. **Should each new module-hygiene primitive land with a new
   `docs/examples/0N-*.md` runnable example?** I've planned 08, 09,
   10. That's ~3 new examples + their runnable tests, which means
   ~3 more LocalStack containers per CI run. Fine, or excessive?

7. **Cross-phase rebase strategy.** PRs 2–4 are designed to be
   independent of each other after PR 1 lands. If they land in
   sequence, do PRs 3 and 4 each need to rebase on the latest main
   after the previous merge? Mechanically yes; just flagging the
   sequencing cost.

## References

- [DESIGN-0003 — Module hygiene primitives and per-service package
  layout][design-0003]
- [INV-0002 — EKS coverage via LocalStack][inv-0002]
- [INV-0001 — terratest 1.0 context variant migration][inv-0001] —
  established the paired-method pattern this work preserves
- [IMPL-0003 — terratest 1.0 context migration][impl-0003] — prior
  template for how to structure a multi-phase libtftest implementation
- `aws-sdk-go-v2/service/<name>` — package layout precedent
- claude-skills issue #53 — Track 1 (libtftest plugin v0.2.0) merged;
  this work re-opens Track 1 for plugin v0.3.0

[inv-0001]: ../investigation/0001-terratest-10-context-variant-migration.md
[impl-0003]: 0003-terratest-10-context-migration.md
