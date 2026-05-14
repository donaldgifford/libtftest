---
id: IMPL-0004
title: "Module hygiene primitives and per-service package layout"
status: Draft
author: Donald Gifford
created: 2026-05-13
---

<!-- markdownlint-disable-file MD025 MD041 -->

# IMPL 0004: Module hygiene primitives and per-service package layout

**Status:** Draft **Author:** Donald Gifford **Date:** 2026-05-13

<!--toc:start-->
- [Objective](#objective)
- [Scope](#scope)
  - [In Scope](#in-scope)
  - [Out of Scope](#out-of-scope)
- [Versioning Strategy](#versioning-strategy)
- [Branch / Commit Strategy](#branch--commit-strategy)
- [Implementation Phases](#implementation-phases)
  - [Phase 1: assert/ per-service refactor](#phase-1-assert-per-service-refactor)
    - [Tasks](#tasks)
    - [Success Criteria](#success-criteria)
  - [Phase 2: fixtures/ per-service refactor](#phase-2-fixtures-per-service-refactor)
    - [Tasks](#tasks-1)
    - [Success Criteria](#success-criteria-1)
  - [Phase 3: Cross-cutting layout work](#phase-3-cross-cutting-layout-work)
    - [Tasks](#tasks-2)
    - [Success Criteria](#success-criteria-2)
  - [Phase 4: TestCase.AssertIdempotent + AssertIdempotentApply](#phase-4-testcaseassertidempotent--assertidempotentapply)
    - [Tasks](#tasks-3)
    - [Success Criteria](#success-criteria-3)
  - [Phase 5: assert/tags package](#phase-5-asserttags-package)
    - [Tasks](#tasks-4)
    - [Success Criteria](#success-criteria-4)
  - [Phase 6: assert/snapshot package](#phase-6-assertsnapshot-package)
    - [Tasks](#tasks-5)
    - [Success Criteria](#success-criteria-5)
  - [Phase 7: tools/docgen/ marker scanner + feature matrix](#phase-7-toolsdocgen-marker-scanner--feature-matrix)
    - [Tasks](#tasks-6)
    - [Success Criteria](#success-criteria-6)
  - [Phase 8: claude-skills plugin sync](#phase-8-claude-skills-plugin-sync)
    - [Tasks](#tasks-7)
    - [Success Criteria](#success-criteria-7)
  - [Phase 9: Release verification](#phase-9-release-verification)
    - [Tasks](#tasks-8)
    - [Success Criteria](#success-criteria-8)
- [File Changes](#file-changes)
  - [Created](#created)
  - [Modified](#modified)
  - [Deleted](#deleted)
- [Testing Plan](#testing-plan)
- [Dependencies](#dependencies)
- [Resolved Questions](#resolved-questions)
- [Future Work](#future-work)
- [References](#references)
<!--toc:end-->

## Objective

Execute the design defined in DESIGN-0003 — a per-service package layout
refactor plus three module-hygiene primitives (`AssertIdempotent` + double-Apply
variant, `assert/tags`, `assert/snapshot`), plus the matching `claude-skills`
plugin update — as a single feature branch landing one `v0.2.0` minor release.

**Implements:** [DESIGN-0003][design-0003] (originated from
[INV-0002][inv-0002])

[design-0003]:
  ../design/0003-module-hygiene-primitives-and-per-service-package-layout.md
[inv-0002]: ../investigation/0002-eks-coverage-via-localstack.md
[inv-0003]:
  ../investigation/0003-package-documentation-convention-and-gomarkdoc-toolchain.md
[inv-0004]: ../investigation/0004-pro-and-oss-feature-matrix-tooling.md

## Scope

### In Scope

- Per-service sub-packages under `assert/` and `fixtures/`
- `package`-level functions replacing zero-size-struct namespacing
- `TestCase.AssertIdempotent` / `AssertIdempotentContext` /
  `AssertIdempotentApply` / `AssertIdempotentApplyContext`
- New `assert/tags` package: `PropagatesFromRoot` / `PropagatesFromRootContext`
- New `assert/snapshot` package: `JSONStrict`, `JSONStructural`,
  `ExtractIAMPolicies`, `ExtractResourceAttribute`
- New `internal/testfake` package for shared `fakeTB` reuse across per-service
  test files
- All `docs/examples/*.md` and the runnable `examples_integration_test.go`
  updated to the new import shape
- Local Claude Code skills (`.claude/skills/libtftest-add-assertion`,
  `libtftest-add-fixture`) updated to emit the new shape
- claude-skills plugin (`tftest:add-test`, `tftest:add-assertion`,
  `tftest:add-fixture`, `tftest:scaffold`) updated to emit the new shape; plugin
  version bump 0.2.0 → 0.3.0; pin range bump
- libtftest version bump per [Versioning Strategy](#versioning-strategy)

### Out of Scope

- Refactoring `awsx/` — stays flat per DESIGN-0003 rationale
- Backwards-compatibility shim layer or re-exports
- Migrating consumer call sites in any consumer repo
- Auto-generating IAM snapshots — caller produces JSON via
  `terraform show -json`; `assert/snapshot` only compares
- An `assert/eks` package — wait for a real consumer use case
- The `infrastructure-as-code` plugin's generic terratest skill update (tracked
  separately under claude-skills issue #53 Track 2)

## Versioning Strategy

All four parts ship in a single `minor` bump: `v0.1.1` → `v0.2.0`.

Under pre-1.0 SemVer (header in `CHANGELOG.md`) we don't need to split additive
features into their own minor tags — the breaking layout change already forces a
minor bump, and the three additive primitives (idempotency, tags, snapshot) ride
along in the same release. Once we cross v1.0 we'll revisit and require strict
SemVer per public-surface addition.

Plugin manifest version (`plugins/libtftest` in `donaldgifford/claude-skills`)
bumps independently: 0.2.0 → 0.3.0, pin range `>=0.2.0, <1.0.0` →
`>=0.2.0, <1.0.0` (no change — still covers the new tag).

## Branch / Commit Strategy

**One feature branch, one PR, multiple commits, one release.**

- Branch: `inv/eks-localstack-coverage` (current branch carrying INV-0002 +
  DESIGN-0003 + IMPL-0004) is the working branch for the implementation phases
  as well.
- Each phase lands as one or more conventional commits on this branch — no
  rebase between phases.
- The PR opens once all 9 phases are complete; CI's `Bump Version` job consumes
  the `minor` label and produces `v0.2.0`.
- Plugin sync (Phase 8) lands as a separate PR in the
  `donaldgifford/claude-skills` repo because it lives in a different repository.

| Phase | Commit type             | Notes                                                                                      |
| ----- | ----------------------- | ------------------------------------------------------------------------------------------ |
| 1     | `refactor(assert)`      | per-service package split                                                                  |
| 2     | `refactor(fixtures)`    | per-service package split                                                                  |
| 3     | `docs`                  | examples, README, CLAUDE.md, doc.go rollout                                                |
| 4     | `feat(libtftest)`       | `AssertIdempotent` + `AssertIdempotentApply`                                               |
| 5     | `feat(assert/tags)`     | RGT-backed tag propagation                                                                 |
| 6     | `feat(assert/snapshot)` | JSON strict/structural + extraction helpers                                                |
| 7     | `feat(tools/docgen)`    | marker scanner + feature matrix + CI gate (lands under the new `Tooling` cliff.toml group) |
| 8     | (separate repo PR)      | claude-skills plugin v0.3.0                                                                |
| 9     | (cross-cutting)         | release verification after merge                                                           |

## Implementation Phases

Each phase builds on the previous. A phase is complete when all its tasks are
checked off and its success criteria are met.

---

### Phase 1: `assert/` per-service refactor

Move each `assert/<service>.go` to its own per-service package. Drop the
zero-size-struct namespacing pattern in favor of package-level functions.
Preserve the paired `Foo` / `FooContext` shape from INV-0001.

The shared `fakeTB` stub currently lives in `assert/assert_test.go` and is
referenced by every per-service `*Context_PropagatesCancel` test. Move it to a
new `internal/testfake` package so each per-service test file can import it
without duplication.

#### Tasks

- [x] Create `internal/testfake/testfake.go` with the existing `fakeTB` minimal
      `testing.TB` surface (Helper, Errorf, Error, Fatalf, Fatal, Skip, Skipf,
      SkipNow, Logf, Log, Context)
- [x] Create `assert/s3/s3.go` (`package s3`) with `BucketExists`,
      `BucketExistsContext`, `BucketHasEncryption`,
      `BucketHasEncryptionContext`, `BucketHasVersioning`,
      `BucketHasVersioningContext`, `BucketBlocksPublicAccess`,
      `BucketBlocksPublicAccessContext`, `BucketHasTag`, `BucketHasTagContext`
      as package-level functions
- [x] Create `assert/s3/s3_test.go` with the existing S3 test coverage using
      `internal/testfake.NewFakeTB(...)`
- [x] Create `assert/dynamodb/dynamodb.go` (`package dynamodb`) with
      `TableExists`, `TableExistsContext`
- [x] Create `assert/dynamodb/dynamodb_test.go`
- [x] Create `assert/iam/iam.go` (`package iam`) with `RoleExists`,
      `RoleExistsContext`, `RoleHasInlinePolicy`, `RoleHasInlinePolicyContext`
      (preserve `libtftest.RequirePro(tb)` gates)
- [x] Create `assert/iam/iam_test.go`
- [x] Create `assert/ssm/ssm.go` (`package ssm`) with `ParameterExists`,
      `ParameterExistsContext`, `ParameterHasValue`, `ParameterHasValueContext`
- [x] Create `assert/ssm/ssm_test.go`
- [x] Create `assert/lambda/lambda.go` (`package lambda`) with `FunctionExists`,
      `FunctionExistsContext`
- [x] Create `assert/lambda/lambda_test.go`
- [x] Delete `assert/s3.go`, `assert/dynamodb.go`, `assert/iam.go`,
      `assert/ssm.go`, `assert/lambda.go`
- [x] Delete `assert/assert.go` if it only held the zero-size struct vars
      (`var S3 = s3Asserts{}` etc.); keep otherwise
- [x] Delete `assert/assert_test.go` once `fakeTB` is migrated and every
      per-service file has its coverage
- [x] Add `assert/s3/doc.go`, `assert/dynamodb/doc.go`, `assert/iam/doc.go`,
      `assert/ssm/doc.go`, `assert/lambda/doc.go`, and
      `internal/testfake/doc.go` — one per new package, each containing only the
      `package <name>` declaration and a multi-paragraph godoc-compliant package
      comment (per the [INV-0003][inv-0003] convention now adopted repo-wide)
- [x] Add `// libtftest:requires pro <reason>` markers on `assert/iam` functions
      that call `libtftest.RequirePro(tb)` (per the [INV-0004][inv-0004] marker
      convention)

#### Success Criteria

- `go build ./...` succeeds
- `go test ./assert/...` passes
- No file remains at `assert/<service>.go` (top-level)
- `grep -rn "var S3 = s3Asserts" .` returns zero hits
- `grep -rn "assert.S3.BucketExists" .` returns zero hits in source (matches in
  markdown docs are addressed in Phase 3)

---

### Phase 2: `fixtures/` per-service refactor

Move the single `fixtures/fixtures.go` into per-service packages with short
function names (the package name carries the service prefix).

#### Tasks

- [x] Create `fixtures/s3/s3.go` (`package s3`) with `SeedObject`,
      `SeedObjectContext`
- [x] Create `fixtures/s3/s3_test.go` carrying forward the existing S3 fixture
      cancellation + cleanup-registered tests
- [x] Create `fixtures/ssm/ssm.go` (`package ssm`) with `SeedParameter`,
      `SeedParameterContext`
- [x] Create `fixtures/ssm/ssm_test.go`
- [x] Create `fixtures/secretsmanager/secretsmanager.go`
      (`package secretsmanager`) with `SeedSecret`, `SeedSecretContext`
- [x] Create `fixtures/secretsmanager/secretsmanager_test.go`
- [x] Create `fixtures/sqs/sqs.go` (`package sqs`) with `SeedMessage`,
      `SeedMessageContext`
- [x] Create `fixtures/sqs/sqs_test.go`
- [x] Each per-service test imports `internal/testfake`
- [x] Delete `fixtures/fixtures.go` and `fixtures/fixtures_test.go`
- [x] Verify `context.WithoutCancel(ctx)` cleanup pattern survives the move

#### Success Criteria

- `go build ./...` succeeds
- `go test ./fixtures/...` passes
- No file remains at `fixtures/fixtures.go`
- `grep -rn "fixtures.SeedS3Object" .` returns zero hits in source

---

### Phase 3: Cross-cutting layout work

Update everything that referenced the old layout: docs, examples, local skill
templates, internal callers. After Phase 3, the layout refactor is fully
self-contained and the additive primitives (Phases 4–6) can be added without
touching call sites again.

#### Tasks

- [x] `grep -rn 'assert\.\(S3\|DynamoDB\|IAM\|SSM\|Lambda\)\.' .` — enumerate
      every remaining call site (likely only in docs + examples after Phases
      1–2)
- [x] `grep -rn 'fixtures\.Seed' .` — enumerate every remaining seed-call call
      site
- [x] Update `docs/examples/01-basic-s3-test.md` to use new import shape
      (`s3assert`, `s3fix`)
- [x] Update `docs/examples/03-plan-testing.md`
- [x] Update `docs/examples/04-fixtures.md`
- [x] Update `docs/examples/07-cancellation.md`
- [x] Update `docs/examples/README.md` if it has API surface examples
- [x] Update `docs/examples/examples_integration_test.go` — regenerate runnable
      tests against the new layout; verify they still compile under
      `//go:build integration_examples`
- [x] Update `README.md` "Features", "Quick Start", "Package Overview", and any
      other API-surface sections
- [x] Update `CLAUDE.md` status line + Context API surface section
- [x] Update `.claude/skills/libtftest-add-assertion/SKILL.md` to describe the
      new shape
- [x] Update
      `.claude/skills/libtftest-add-assertion/references/assertion-template.go.tmpl`
      — emit `package <service>` + package-level functions instead of zero-size
      struct + methods
- [x] Update `.claude/skills/libtftest-add-fixture/SKILL.md`
- [x] Update
      `.claude/skills/libtftest-add-fixture/references/fixture-template.go.tmpl`
- [x] **Repo-wide `doc.go` rollout** (per [INV-0003][inv-0003]): lift the
      existing `// Package <name>` comment from its current home (e.g.
      `assert.go`, `config.go`, `workspace.go`) into a dedicated `doc.go` for
      every pre-existing package, and expand the comment to a multi-paragraph
      godoc-compliant explanation of package purpose. Packages: `assert/`
      (deprecated top-level doc — leave a `// Package assert is deprecated.`
      note pointing to `assert/<service>/`), `awsx/` (the deliberate-flat-layout
      note already drafted), `fixtures/` (same deprecation note), `harness/`,
      `internal/dockerx/`, `internal/logx/`, `internal/naming/`, `localstack/`,
      `sneakystack/`, `sneakystack/services/`, `tf/`, `cmd/libtftest/`,
      `cmd/sneakystack/`
- [x] After the `doc.go` rollout, remove the `// Package <name>` comment from
      its previous home so it's not duplicated
- [x] Update `CLAUDE.md` Code Conventions section to list the
      `doc.go`-per-package rule and the `// libtftest:requires <tag>...` marker
      rule (already drafted)
- [x] Run `claudelint run .claude/` clean (or verify the CI `skills.yml` job
      stays green if claudelint is not in the local toolchain)
- [x] Run `make fmt` and `make lint` clean
- [x] Run `make ci` clean (lint + test + build + license-check)

#### Success Criteria

- All `docs/examples/` markdown uses the new import shape
- `go test -tags=integration_examples -v ./docs/examples/...` passes locally (or
  in CI if Docker isn't available)
- Local skill templates emit per-service-package code by default
- `make ci` clean
- Every Go package in the repo has a `doc.go` and no other file in the package
  carries a `// Package <name>` block:
  `find . -type d -exec test -f {}/doc.go \;` covers every package containing a
  `.go` file (modulo `cmd/` packages which keep package main's documentation in
  `main.go` per Go convention if no `doc.go` exists; we still add one for
  consistency)
- `grep -rn '^// libtftest:requires ' assert/ fixtures/ libtftest.go` catches
  every marker; every function in those files that calls
  `libtftest.RequirePro(tb)` is accompanied by a marker

---

### Phase 4: `TestCase.AssertIdempotent` + `AssertIdempotentApply`

Implement both variants of the idempotency check. Lives in `libtftest.go` next
to the other `TestCase` methods (Apply, Plan, Output) per DESIGN-0003's "lives
on TestCase" rationale.

#### Tasks

- [x] Add `AssertIdempotent()` shim and `AssertIdempotentContext(ctx)` to
      `libtftest.go`
- [x] Add `AssertIdempotentApply()` shim and `AssertIdempotentApplyContext(ctx)`
      to `libtftest.go`
- [x] Doc comments must end with periods (godot linter); each shim must have a
      `// <Name> is a shim that calls <Name>Context with     tb.Context().` line
- [x] Add a unit test confirming both variants are wired (compile-time
      method-signature check, in the libtftest_test.go style)
- [x] Add `TestAssertIdempotent_S3Module` integration test in
      `libtftest_integration_test.go` — happy path (idempotent S3 module passes
      both variants)
- [x] Add `TestAssertIdempotent_DetectsDrift` integration test — injects
      synthetic drift via `terraform_data` with a `timestamp()` input (built-in
      Terraform resource, no provider download), asserts the check fails
- [x] Update `docs/examples/` with a new `08-idempotency.md` example + matching
      `Test_Example08_Idempotency` in `examples_integration_test.go`
- [x] Update `docs/examples/README.md` index
- [x] Update `README.md` Features list to mention idempotency assertions
- [x] Update `CLAUDE.md` Context API surface section
- [x] Run `make ci` clean

#### Success Criteria

- Both variants compile and have doc comments
- Integration tests pass against LocalStack (`make test-integration`)
- Synthetic drift test fails the assertion as expected

---

### Phase 5: `assert/tags` package

Service-agnostic tag propagation assertion backed by the AWS Resource Groups
Tagging API (`resourcegroupstaggingapi.GetResources`).

#### Tasks

- [x] Add `NewResourceGroupsTagging(cfg aws.Config)` constructor to
      `awsx/clients.go` (kept in the existing flat-layout bag-of-constructors
      per `awsx/doc.go`'s deliberate-flat-layout note, instead of a new
      per-service file — both shapes are consistent with the deliberate-flat
      decision in DESIGN-0003 Resolved Question 1)
- [x] Create `assert/tags/tags.go` (`package tags`) with
      `PropagatesFromRoot(tb, cfg, baseline, arns...)` and
      `PropagatesFromRootContext(tb, ctx, cfg, baseline, arns...)`
- [x] Implement subset-check semantics: every key/value in `baseline` must be
      present on every ARN; extra tags on the resource are allowed
- [x] Collect errors across all ARNs before calling `tb.Errorf` — surface
      "resource X is missing tag Y" + "resource X has tag Y with value Z,
      expected W" all at once
- [x] Verify LocalStack OSS support: integration test path enables the
      `resourcegroupstaggingapi` service in `Options.Services`; the
      authoritative deterministic coverage lives in
      `assert/tags/tags_test.go` (failure-path unit tests via
      `internal/testfake`)
- [x] Create `assert/tags/tags_test.go` with unit tests via `internal/testfake`
      covering: missing key, wrong value, multiple-ARN aggregation, cancellation
      propagation
- [x] Add `assert/tags` integration test in `libtftest_integration_test.go`
      using `testdata/mod-tagged/` (two tagged S3 buckets); compile-time
      surface guard for the public entry points
- [x] Update `docs/examples/` with a new `09-tag-propagation.md` example +
      matching runnable test
- [x] Update `docs/examples/README.md` index
- [x] Update `README.md` Features list
- [x] Run `make ci` clean

#### Success Criteria

- `assert/tags` package compiles and tests pass
- Integration test against LocalStack passes (OSS, sneakystack mock, or Pro path
  per the decision tree above)

---

### Phase 6: `assert/snapshot` package

Generic JSON snapshot testing with an `UPDATE_SNAPSHOTS=1` rewrite protocol,
plus IAM-specific and general-purpose extraction helpers for Terraform plan
JSON.

#### Tasks

- [x] Create `assert/snapshot/snapshot.go` (`package snapshot`) with
      `JSONStrict(tb, actual, path)` and `JSONStructural(tb, actual, path)`
- [x] Implement structural normalization: recursively sort keys, strip
      insignificant whitespace, normalize numeric types where JSON's spec is
      ambiguous
- [x] Wire `LIBTFTEST_UPDATE_SNAPSHOTS=1` rewrite protocol — on mismatch,
      overwrite `path` with `actual` and pass the test; log via `tb.Logf` so CI
      runs surface what was overwritten
- [x] Implement `ExtractIAMPolicies(planJSON []byte) (map[string][]byte, error)`
      — walks `planned_values.root_module.resources` for `aws_iam_role`,
      `aws_iam_policy`, `aws_iam_role_policy`, `aws_iam_role_policy_attachment`;
      returns one entry per role per policy keyed by
      `<resource_address>.<assume_role|inline:<name>|managed:<arn>|policy>`
- [x] Implement `ExtractResourceAttribute(planJSON, addr, path) ([]byte, error)`
      — generic JSON path extraction under
      `planned_values.root_module.resources[?address==addr].values.<path>`
- [x] Create `assert/snapshot/snapshot_test.go` covering: identical JSON,
      byte-different-but-structurally-equal (strict fails, structural passes),
      missing snapshot file (without update mode → fail; with update mode →
      write + pass), structurally different JSON (both forms fail)
- [x] Create `assert/snapshot/extract_test.go` covering: extract IAM policies
      from a fixture plan JSON, extract a KMS key policy via the generic helper,
      missing resource address (returns error)
- [x] Generate fixture plan JSON for tests:
      `assert/snapshot/testdata/plan-iam-kms.json` — hand-crafted minimal
      `terraform show -json` payload covering an IAM role with
      assume-role + inline + managed attachment, a standalone IAM policy, a
      KMS key, and a tagged S3 bucket (no external Terraform dependency
      required to run the tests)
- [x] Update `docs/examples/` with a new `10-snapshot-iam.md` example +
      matching `Test_Example10_SnapshotIAM` in `examples_integration_test.go`
- [x] Update `docs/examples/README.md` index
- [x] Update `README.md` Features list
- [x] Run `make ci` clean

**Determinism note (managed policies).** `ExtractIAMPolicies` must produce a
deterministic output. Inline policies are extracted as full JSON document
strings. AWS managed policy attachments
(`aws_iam_role_policy_attachment.policy_arn`) are emitted as the canonical ARN
string (e.g. `arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess`). We **do not**
fetch the live document for AWS-managed policies — those ARNs are effectively an
enum (AWS owns them, they don't drift in test-relevant ways), and fetching them
would make the helper network-dependent and non-deterministic. Customer-managed
policies attached by ARN render as the ARN string for the same reason: the
snapshot tests propagation of the attachment, not the policy document itself
(that's a separate snapshot if the customer managed policy lives in the module).

#### Success Criteria

- `assert/snapshot` package compiles and tests pass (no LocalStack required)
- Extraction helpers correctly parse the fixture plan JSON
- `LIBTFTEST_UPDATE_SNAPSHOTS=1` mode works locally
- `ExtractIAMPolicies` output is deterministic across runs (no network calls, no
  map-iteration sensitivity)

---

### Phase 7: `tools/docgen/` marker scanner + feature matrix

Implement the docgen tool that consumes the
`// libtftest:requires <tag>[,<tag>...] <reason>` markers (added in Phase 1 and
the Phase 3 rollout) and renders a feature matrix to `docs/feature-matrix.md`.
Add a `make check-markers` CI gate that fails when a function calls
`libtftest.RequirePro(tb)` without an accompanying marker. Per
[INV-0004][inv-0004] (Concluded).

The tool is intentionally regex-based — it does NOT import any `libtftest`
packages, walks source files only, and stays version-agnostic.

#### Tasks

- [x] Create `tools/docgen/main.go` (`package main`) with a `scan` subcommand
      that: walks the repo's `.go` files (skipping `.git`, `vendor`, `build`,
      `.claude`, `node_modules`, `.docz`); parses each file with `go/parser` +
      `go/ast`; pairs each `// libtftest:requires <tags> <reason>` doc comment
      line with the function the doc is attached to; emits a stable-sorted
      JSON intermediate representation (function, receiver, package, tags,
      reason, source file/line)
- [x] Add a `render` subcommand that consumes the JSON IR (or runs `scan`
      itself when neither `-in` nor a piped stdin is provided) and writes
      `docs/feature-matrix.md` — one row per marker, with a summary block
      counting per-tag totals, pipe characters in the Reason column escaped,
      and tags rendered as sorted bold tokens
- [x] Add a `check` subcommand that walks the repo (skipping `_test.go`
      files), inspects every `*ast.FuncDecl`, and fails with `file:line` when
      a function calls `libtftest.RequirePro(` without a `// libtftest:requires`
      marker on its doc. Functions with a marker but no `RequirePro` call are
      permitted — markers may anticipate future gates
- [x] Add `make docs-matrix` target that runs `tools/docgen render`
- [x] Add `make check-markers` target that runs `tools/docgen check`
- [x] Wire `make check-markers` into `make ci`
- [x] Add `tools/docgen/main_test.go` with table-driven tests covering:
      single-tag marker, multi-tag marker, missing marker (caught by `check`),
      function with marker but no `RequirePro` call (allowed), test files
      ignored by `check`, stable sort order across runs, render determinism,
      pipe escaping, multi-tag sort
- [x] Add `tools/doc.go` and `tools/docgen/doc.go` documenting the layout
- [x] Run `tools/docgen render` and commit the initial `docs/feature-matrix.md`
      (4 markers, all currently on assert/iam functions)
- [x] Update `README.md` to link to `docs/feature-matrix.md`
- [x] Update `CLAUDE.md` to mention the `tools/docgen` location

#### Success Criteria

- `tools/docgen` binary builds and tests pass
- `make check-markers` exits zero against the current tree (every `RequirePro`
  caller has a marker)
- `make docs-matrix` regenerates `docs/feature-matrix.md` deterministically
  (same input → same output, run-to-run)
- `docs/feature-matrix.md` exists, lists every marker function, with `pro` (and
  any other) tags rendered as columns
- `make ci` includes `check-markers` and stays green

---

### Phase 8: claude-skills plugin sync

Bump the consumer-facing plugin to track the new libtftest layout and feature
set. Mirrors the work done for v0.1.0 in `feat/libtftest-plugin-v0.2.0`.

> **Scope note.** Phase 8 lands as a separate PR in the
> `donaldgifford/claude-skills` repo per the Branch / Commit Strategy
> table above (`feat(plugin)` / `chore(plugin)` commits never appear on
> this libtftest branch). Implemented on branch
> `feat/libtftest-plugin-v0.3.0` in `donaldgifford/claude-skills`; the
> libtftest IMPL-0004 branch tracks these tasks for cross-repo
> coordination only.

#### Tasks

- [x] Bump `plugins/libtftest/.claude-plugin/plugin.json` version 0.2.0 → 0.3.0
- [x] Matching bump in `.claude-plugin/marketplace.json`
- [x] Version-pin range across all `tftest:*` skill bodies, `_version-check.md`,
      `_frontmatter.md`, `README.md`, and the reviewer agent: `>=0.1.0, <1.0.0`
      → `>=0.2.0, <1.0.0`
- [x] Update `tftest:add-test` SKILL.md + scaffold to use the new import shape
- [x] Update `tftest:add-assertion` SKILL.md + scaffold to use the new
      per-service-package shape (`s3assert.BucketExists`, etc.)
- [x] Update `tftest:add-fixture` SKILL.md + scaffold to use the new
      per-service-package shape (`s3fix.SeedObject`, etc.)
- [x] Update `tftest:scaffold` (single-layout template) to use the new import
      shape; `AssertIdempotent` mentioned as a module-hygiene convention
- [x] Update umbrella `tftest` SKILL.md to surface the new module-hygiene
      primitives (idempotency, tags, snapshot) + feature-matrix pointer
- [x] Update `plugins/libtftest/CHANGELOG.md` with a `[0.3.0]` entry
- [x] Update `plugins/libtftest/README.md` API surface tables
- [x] Run `make test-plugin PLUGIN=libtftest` clean
- [x] Run `scripts/sync_readme.py` clean
- [x] Run `git-cliff -o CHANGELOG.md` and commit the result as
      `chore(changelog): ...`

#### Success Criteria

- Plugin manifest at version 0.3.0
- Pin range points at libtftest minor that ships these features
- All `tftest:*` skills emit per-service-package code by default
- Plugin tests + sync-readme clean
- Separate PR ready to open in the `claude-skills` repo

---

### Phase 9: Release verification

Cross-cutting verification that runs once the PR opens and again after merge.
Not a separate PR.

> **Scope note.** Every Phase 9 task is post-PR-open or post-merge —
> they cannot be checked from this branch before the PR exists. The
> libtftest IMPL-0004 implementation work concludes at the end of
> Phase 7; Phase 9 lives here as the merge checklist.

#### Tasks

- [x] PR CI green (lint, test, integration, docker, drift check, claudelint)
- [ ] PR merges to `main` with the `minor` label
- [ ] `Bump Version` + `Release` + `Changelog Sync` + `Docker` workflow jobs all
      green on the post-merge run
- [ ] `v0.2.0` tag + GitHub Release published with goreleaser notes
- [ ] Multi-arch `sneakystack` image at
      `ghcr.io/donaldgifford/sneakystack:0.2.0` signed via cosign keyless
- [ ] Plugin sync PR (Phase 8) in `donaldgifford/claude-skills` lands; plugin
      v0.3.0 published; pin range covers libtftest `>=0.2.0, <1.0.0`
- [x] No `chore(deps)` dependabot PRs left orphaned
- [ ] `CHANGELOG.md` on `main` reflects the v0.2.0 section produced by
      `git-cliff` without any manual fixups
- [x] Update memory `MEMORY.md` pointer to a new memory entry summarizing the
      layout-change shape (`impl-0004-shape.md`)

#### Success Criteria

- One libtftest tag exists: `v0.2.0`
- Plugin manifest at version 0.3.0
- GitHub Release published for `v0.2.0` with goreleaser-rendered notes
- Signed multi-arch `ghcr.io/donaldgifford/sneakystack:0.2.0` image
- IMPL-0004 doc status flipped to `Completed`
- DESIGN-0003 doc status flipped to `Accepted`
- INV-0002 doc status flipped to `Concluded`

---

## File Changes

### Created

| File                                                     | Purpose                                                                           |
| -------------------------------------------------------- | --------------------------------------------------------------------------------- |
| `internal/testfake/testfake.go`                          | Shared `fakeTB` for per-service test packages                                     |
| `assert/s3/s3.go` + `_test.go`                           | Migrated S3 assertions                                                            |
| `assert/dynamodb/dynamodb.go` + `_test.go`               | Migrated DynamoDB assertions                                                      |
| `assert/iam/iam.go` + `_test.go`                         | Migrated IAM assertions                                                           |
| `assert/ssm/ssm.go` + `_test.go`                         | Migrated SSM assertions                                                           |
| `assert/lambda/lambda.go` + `_test.go`                   | Migrated Lambda assertions                                                        |
| `assert/tags/tags.go` + `_test.go`                       | `PropagatesFromRoot` + ctx variant                                                |
| `assert/snapshot/snapshot.go` + `_test.go`               | JSON snapshot diffing                                                             |
| `assert/snapshot/extract.go` + `_test.go`                | IAM + generic plan-JSON extraction                                                |
| `assert/snapshot/testdata/plan-iam-kms.json`             | Fixture plan JSON                                                                 |
| `awsx/resourcegroupstaggingapi.go`                       | New constructor for tags backend                                                  |
| `awsx/doc.go`                                            | Package-level godoc explaining the deliberate flat layout                         |
| `assert/doc.go`                                          | Deprecated top-level package note pointing to per-service sub-packages            |
| `assert/s3/doc.go`                                       | Per-service package comment                                                       |
| `assert/dynamodb/doc.go`                                 | Per-service package comment                                                       |
| `assert/iam/doc.go`                                      | Per-service package comment + Pro-only note                                       |
| `assert/ssm/doc.go`                                      | Per-service package comment                                                       |
| `assert/lambda/doc.go`                                   | Per-service package comment                                                       |
| `assert/tags/doc.go`                                     | Package comment for new tags package                                              |
| `assert/snapshot/doc.go`                                 | Package comment for new snapshot package                                          |
| `fixtures/doc.go`                                        | Deprecated top-level package note pointing to per-service sub-packages            |
| `fixtures/s3/doc.go`                                     | Per-service package comment                                                       |
| `fixtures/ssm/doc.go`                                    | Per-service package comment                                                       |
| `fixtures/secretsmanager/doc.go`                         | Per-service package comment                                                       |
| `fixtures/sqs/doc.go`                                    | Per-service package comment                                                       |
| `harness/doc.go`                                         | Package comment lifted from existing inline source                                |
| `internal/dockerx/doc.go`                                | Package comment lifted from existing inline source                                |
| `internal/logx/doc.go`                                   | Package comment lifted from existing inline source                                |
| `internal/naming/doc.go`                                 | Package comment lifted from existing inline source                                |
| `internal/testfake/doc.go`                               | Package comment for new testfake package                                          |
| `localstack/doc.go`                                      | Package comment lifted from existing inline source                                |
| `sneakystack/doc.go`                                     | Package comment lifted from existing inline source                                |
| `sneakystack/services/doc.go`                            | Package comment lifted from existing inline source                                |
| `tf/doc.go`                                              | Package comment lifted from existing inline source                                |
| `cmd/libtftest/doc.go`                                   | Package main comment                                                              |
| `cmd/sneakystack/doc.go`                                 | Package main comment                                                              |
| `tools/docgen/main.go`                                   | Marker scanner + matrix renderer + CI check (`scan`/`render`/`check` subcommands) |
| `tools/docgen/main_test.go`                              | Table-driven coverage                                                             |
| `tools/docgen/doc.go`                                    | Package main comment                                                              |
| `tools/doc.go`                                           | Directory-purpose comment                                                         |
| `docs/feature-matrix.md`                                 | Rendered Pro/OSS/mockta/etc. matrix (generated by `make docs-matrix`)             |
| `fixtures/s3/s3.go` + `_test.go`                         | Migrated S3 fixtures                                                              |
| `fixtures/ssm/ssm.go` + `_test.go`                       | Migrated SSM fixtures                                                             |
| `fixtures/secretsmanager/secretsmanager.go` + `_test.go` | Migrated Secrets Manager fixtures                                                 |
| `fixtures/sqs/sqs.go` + `_test.go`                       | Migrated SQS fixtures                                                             |
| `docs/examples/08-idempotency.md`                        | Idempotency example                                                               |
| `docs/examples/09-tag-propagation.md`                    | Tag propagation example                                                           |
| `docs/examples/10-snapshot-iam.md`                       | Snapshot IAM example                                                              |

### Modified

| File                                                           | Reason                                                       |
| -------------------------------------------------------------- | ------------------------------------------------------------ |
| `libtftest.go`                                                 | Add `AssertIdempotent` family (4 methods)                    |
| `libtftest_integration_test.go`                                | Add idempotency + drift detection tests                      |
| `docs/examples/01-basic-s3-test.md`                            | New import shape                                             |
| `docs/examples/03-plan-testing.md`                             | New import shape                                             |
| `docs/examples/04-fixtures.md`                                 | New import shape                                             |
| `docs/examples/07-cancellation.md`                             | New import shape                                             |
| `docs/examples/README.md`                                      | Index Phase 4–6 examples                                     |
| `docs/examples/examples_integration_test.go`                   | Add tests for examples 8–10; update existing for new shape   |
| `README.md`                                                    | Features list + Quick Start + Package Overview               |
| `CLAUDE.md`                                                    | Status line + Context API surface                            |
| `CHANGELOG.md`                                                 | Regenerated by git-cliff after each phase                    |
| `cliff.toml`                                                   | Add `Tooling` group for `feat(tools)`/`chore(tools)` commits |
| `Makefile`                                                     | New `docs-matrix` + `check-markers` targets; wire into `ci`  |
| `.claude/skills/libtftest-add-assertion/SKILL.md` + template   | Emit per-service-package shape                               |
| `.claude/skills/libtftest-add-fixture/SKILL.md` + template     | Emit per-service-package shape                               |
| (claude-skills) `plugins/libtftest/.claude-plugin/plugin.json` | Version 0.3.0                                                |
| (claude-skills) `.claude-plugin/marketplace.json`              | Version 0.3.0                                                |
| (claude-skills) `plugins/libtftest/skills/*/SKILL.md`          | Pin range + new import shape                                 |
| (claude-skills) `plugins/libtftest/skills/tftest-scaffold/**`  | Template emits new shape                                     |
| (claude-skills) `plugins/libtftest/CHANGELOG.md`               | `[0.3.0]` entry                                              |
| (claude-skills) `plugins/libtftest/README.md`                  | API surface tables                                           |

### Deleted

| File                                                           | Reason                                                                   |
| -------------------------------------------------------------- | ------------------------------------------------------------------------ |
| `assert/s3.go`, `dynamodb.go`, `iam.go`, `ssm.go`, `lambda.go` | Migrated to per-service packages                                         |
| `assert/assert.go`                                             | Zero-size struct vars are gone (if file held only those)                 |
| `assert/assert_test.go`                                        | `fakeTB` migrated to `internal/testfake`; per-service tests own coverage |
| `fixtures/fixtures.go`                                         | Migrated to per-service packages                                         |
| `fixtures/fixtures_test.go`                                    | Per-service tests own coverage                                           |

## Testing Plan

- [x] Unit test: each new `assert/<service>` package mirrors the coverage the
      pre-refactor file had — at minimum the `*Context_PropagatesCancel` test
      per assertion
- [x] Unit test: each new `fixtures/<service>` package covers the cancellation +
      `WithoutCancel` cleanup pattern from INV-0001
- [x] Unit test: `assert/tags` covers missing-key, wrong-value, multi-ARN
      aggregation, ctx propagation
- [x] Unit test: `assert/snapshot` covers identical / structurally- equal /
      different / missing-file / update-mode scenarios for both strict and
      structural variants
- [x] Unit test: `assert/snapshot.ExtractIAMPolicies` against fixture plan JSON
      containing `aws_iam_role` + `aws_iam_policy`
- [x] Unit test: `assert/snapshot.ExtractResourceAttribute` against a non-IAM
      resource type (KMS key)
- [x] Integration test: end-to-end `TestCase.AssertIdempotent` against a
      known-idempotent S3 module (Plan-only path; Apply blocked by LocalStack
      4.4 S3 CreateBucket MalformedXML — same caveat as the existing
      TestNew_FullLifecycle case)
- [x] Integration test: `TestCase.AssertIdempotentApply` wiring exercised via
      compile-time signature guard in `TestContextMethodSignatures`; the
      runtime double-Apply path inherits the same Apply caveat
- [x] Integration test: synthetic drift causes `AssertIdempotent` to fail (via
      `testdata/mod-drifting/` + a substituted FakeTB)
- [x] Integration test: `assert/tags.PropagatesFromRoot` against a tagged
      two-bucket module (`testdata/mod-tagged/`); deterministic comparison
      coverage in `assert/tags/tags_test.go`
- [x] `make ci` green on every PR (includes the new `check-markers` gate)
- [x] `make test-coverage` shows no coverage regression
- [x] `make test-examples` (Docker required) — every new `examples/0N-*.md`
      has a matching `Test_ExampleNN_*` function
- [x] CHANGELOG drift check green on every PR

## Dependencies

- libtftest v0.1.1 (the current latest tag) as the baseline — every PR rebases
  on a stable main
- terratest v1.0.x — no version bump needed; this work doesn't change terratest
  usage
- Go 1.26 — already in `go.mod`
- LocalStack OSS `2026.04.0` (CI default) and Pro `2026.5.0.dev121` locally —
  see INV-0002. The `assert/tags` Resource Groups Tagging API coverage is the
  only EKS-adjacent concern; see [Resolved Question 4](#resolved-questions) for
  the decision tree
- AWS SDK v2 `resourcegroupstaggingapi` client (new direct dep for
  `awsx/resourcegroupstaggingapi.go`)
- claude-skills repo PR landed (Phase 8) before the libtftest `tftest:*` skills
  can be advertised as compatible

## Resolved Questions

1. **Versioning strategy.** _Resolved 2026-05-13._ Ship as **one `minor` bump
   (`v0.2.0`)** covering all four parts. Pre-1.0 SemVer gives us latitude to
   bundle additive features with a breaking layout change. We'll revisit strict
   per-feature minor bumps once we cross v1.0.

2. **`fakeTB` location.** _Resolved 2026-05-13._ Move to `internal/testfake/`.
   If a per-service package surfaces a need to specialise the fake (e.g.
   capturing helper-call counts), we'll split it then — not preemptively.

3. **`awsx/` "deliberate non-change" CHANGELOG marker.** _Resolved 2026-05-13._
   Skip the empty `chore(awsx)` commit. Instead, document package intent via a
   future repo-wide **`doc.go` convention + `gomarkdoc` toolchain** (see
   [Future Work — Item 1](#future-work)). For this PR specifically, we'll add a
   placeholder `awsx/doc.go` with a package-level godoc comment explaining the
   deliberate flat layout — and the rest of the convention rolls out in a
   follow-up INV.

4. **LocalStack support for Resource Groups Tagging API.** _Resolved
   2026-05-13._ Phase 5's verification step picks the path at implementation
   time:
   - If OSS supports it → ship as designed
   - If OSS partially supports it → mock the gap in `sneakystack` (matches the
     existing IAM-IDC / Organizations pattern)
   - If OSS does not support it → gate behind `libtftest.RequirePro` The rule is
     "for API-call gaps, prefer mock-in-sneakystack or `RequirePro` over
     standing up full alternatives".

5. **`ExtractIAMPolicies` return shape.** _Resolved 2026-05-13._ Always favour
   deterministic output. Inline policies render as the full inline JSON
   document. AWS-managed and customer-managed policy attachments render as the
   ARN string (treated as an enum-like identifier — AWS doesn't change the
   well-known managed ARNs in test-relevant ways, and we don't manage them). No
   network calls at extraction time. See the Phase 6 determinism note for the
   full spec.

6. **`docs/examples/` count for the new primitives.** _Resolved 2026-05-13._
   Ship **3 separate examples** (08-idempotency.md, 09-tag-propagation.md,
   10-snapshot-iam.md). The existing pattern is one concept per example file
   (01-basic-s3 through 07-cancellation), each in the 2–5 KB range. Bundling all
   three primitives under a single "module-hygiene" example would break the
   discoverability pattern — consumers cross-link a single example URL when
   teaching a teammate, and a combined doc loses that affordance. The CI cost
   (~3 more LocalStack containers per integration run) is acceptable; the
   existing examples already spin LocalStack per case.

7. **Cross-phase rebase strategy.** _Resolved 2026-05-13._ Single PR, single
   branch, all commits land together. No rebase needed.

## Future Work

These items came up while resolving IMPL-0004's open questions but are out of
scope here. Each gets its own INV before the next IMPL plan is drafted.

1. **`gomarkdoc` rendering toolchain + CI doc.go enforcement.** The
   `doc.go`-per-package _convention_ itself ships as part of this IMPL (Phase 3
   rollout — see [INV-0003][inv-0003], status Concluded). What remains deferred
   is:
   - Wiring [`princjef/gomarkdoc`](https://github.com/princjef/gomarkdoc) (or a
     custom docgen) behind a `make docs` target to render package docs to
     markdown under `docs/api/`
   - A small CI check (`scripts/check-doc-go.sh` or Go program) that fails when
     a package directory lacks a `doc.go`
   - Pushing the gap fixes upstream into the `go-development` plugin as new
     reference files These need their own DESIGN+IMPL cycle and share docgen
     design space with Future Work item 2.

2. **Upstreaming the marker convention.** The marker convention, the
   `tools/docgen` scanner, the `docs/feature-matrix.md` render, and the
   `make check-markers` CI gate _all_ ship as part of this IMPL (Phase 1, Phase
   3, and Phase 7). [INV-0004 is Concluded][inv-0004]. What's left as future
   work is pushing the marker grammar upstream — either into the
   `go-development` plugin as a new reference file, or as a shared convention
   across the donaldgifford toolbox — once it has baked in this repo for a
   release or two.

## References

- [DESIGN-0003 — Module hygiene primitives and per-service package
  layout][design-0003]
- [INV-0002 — EKS coverage via LocalStack][inv-0002]
- [INV-0001 — terratest 1.0 context variant migration][inv-0001] — established
  the paired-method pattern this work preserves
- [IMPL-0003 — terratest 1.0 context migration][impl-0003] — prior template for
  how to structure a multi-phase libtftest implementation
- `aws-sdk-go-v2/service/<name>` — package layout precedent
- claude-skills issue #53 — Track 1 (libtftest plugin v0.2.0) merged; this work
  re-opens Track 1 for plugin v0.3.0

[inv-0001]: ../investigation/0001-terratest-10-context-variant-migration.md
[impl-0003]: 0003-terratest-10-context-migration.md
