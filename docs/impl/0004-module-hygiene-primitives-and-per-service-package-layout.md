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
- [ ] Create `assert/dynamodb/dynamodb.go` (`package dynamodb`) with
      `TableExists`, `TableExistsContext`
- [ ] Create `assert/dynamodb/dynamodb_test.go`
- [ ] Create `assert/iam/iam.go` (`package iam`) with `RoleExists`,
      `RoleExistsContext`, `RoleHasInlinePolicy`, `RoleHasInlinePolicyContext`
      (preserve `libtftest.RequirePro(tb)` gates)
- [ ] Create `assert/iam/iam_test.go`
- [ ] Create `assert/ssm/ssm.go` (`package ssm`) with `ParameterExists`,
      `ParameterExistsContext`, `ParameterHasValue`, `ParameterHasValueContext`
- [ ] Create `assert/ssm/ssm_test.go`
- [ ] Create `assert/lambda/lambda.go` (`package lambda`) with `FunctionExists`,
      `FunctionExistsContext`
- [ ] Create `assert/lambda/lambda_test.go`
- [ ] Delete `assert/s3.go`, `assert/dynamodb.go`, `assert/iam.go`,
      `assert/ssm.go`, `assert/lambda.go`
- [ ] Delete `assert/assert.go` if it only held the zero-size struct vars
      (`var S3 = s3Asserts{}` etc.); keep otherwise
- [ ] Delete `assert/assert_test.go` once `fakeTB` is migrated and every
      per-service file has its coverage
- [ ] Add `assert/s3/doc.go`, `assert/dynamodb/doc.go`, `assert/iam/doc.go`,
      `assert/ssm/doc.go`, `assert/lambda/doc.go`, and
      `internal/testfake/doc.go` — one per new package, each containing only the
      `package <name>` declaration and a multi-paragraph godoc-compliant package
      comment (per the [INV-0003][inv-0003] convention now adopted repo-wide)
- [ ] Add `// libtftest:requires pro <reason>` markers on `assert/iam` functions
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

- [ ] Create `fixtures/s3/s3.go` (`package s3`) with `SeedObject`,
      `SeedObjectContext`
- [ ] Create `fixtures/s3/s3_test.go` carrying forward the existing S3 fixture
      cancellation + cleanup-registered tests
- [ ] Create `fixtures/ssm/ssm.go` (`package ssm`) with `SeedParameter`,
      `SeedParameterContext`
- [ ] Create `fixtures/ssm/ssm_test.go`
- [ ] Create `fixtures/secretsmanager/secretsmanager.go`
      (`package secretsmanager`) with `SeedSecret`, `SeedSecretContext`
- [ ] Create `fixtures/secretsmanager/secretsmanager_test.go`
- [ ] Create `fixtures/sqs/sqs.go` (`package sqs`) with `SeedMessage`,
      `SeedMessageContext`
- [ ] Create `fixtures/sqs/sqs_test.go`
- [ ] Each per-service test imports `internal/testfake`
- [ ] Delete `fixtures/fixtures.go` and `fixtures/fixtures_test.go`
- [ ] Verify `context.WithoutCancel(ctx)` cleanup pattern survives the move

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

- [ ] `grep -rn 'assert\.\(S3\|DynamoDB\|IAM\|SSM\|Lambda\)\.' .` — enumerate
      every remaining call site (likely only in docs + examples after Phases
      1–2)
- [ ] `grep -rn 'fixtures\.Seed' .` — enumerate every remaining seed-call call
      site
- [ ] Update `docs/examples/01-basic-s3-test.md` to use new import shape
      (`s3assert`, `s3fix`)
- [ ] Update `docs/examples/03-plan-testing.md`
- [ ] Update `docs/examples/04-fixtures.md`
- [ ] Update `docs/examples/07-cancellation.md`
- [ ] Update `docs/examples/README.md` if it has API surface examples
- [ ] Update `docs/examples/examples_integration_test.go` — regenerate runnable
      tests against the new layout; verify they still compile under
      `//go:build integration_examples`
- [ ] Update `README.md` "Features", "Quick Start", "Package Overview", and any
      other API-surface sections
- [ ] Update `CLAUDE.md` status line + Context API surface section
- [ ] Update `.claude/skills/libtftest-add-assertion/SKILL.md` to describe the
      new shape
- [ ] Update
      `.claude/skills/libtftest-add-assertion/references/assertion-template.go.tmpl`
      — emit `package <service>` + package-level functions instead of zero-size
      struct + methods
- [ ] Update `.claude/skills/libtftest-add-fixture/SKILL.md`
- [ ] Update
      `.claude/skills/libtftest-add-fixture/references/fixture-template.go.tmpl`
- [ ] **Repo-wide `doc.go` rollout** (per [INV-0003][inv-0003]): lift the
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
- [ ] After the `doc.go` rollout, remove the `// Package <name>` comment from
      its previous home so it's not duplicated
- [ ] Update `CLAUDE.md` Code Conventions section to list the
      `doc.go`-per-package rule and the `// libtftest:requires <tag>...` marker
      rule (already drafted)
- [ ] Run `claudelint run .claude/` clean (or verify the CI `skills.yml` job
      stays green if claudelint is not in the local toolchain)
- [ ] Run `make fmt` and `make lint` clean
- [ ] Run `make ci` clean (lint + test + build + license-check)

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

- [ ] Add `AssertIdempotent()` shim and `AssertIdempotentContext(ctx)` to
      `libtftest.go`
- [ ] Add `AssertIdempotentApply()` shim and `AssertIdempotentApplyContext(ctx)`
      to `libtftest.go`
- [ ] Doc comments must end with periods (godot linter); each shim must have a
      `// <Name> is a shim that calls <Name>Context with     tb.Context().` line
- [ ] Add a unit test confirming both variants are wired (compile-time
      method-signature check, in the libtftest_test.go style)
- [ ] Add `TestAssertIdempotent_S3Module` integration test in
      `libtftest_integration_test.go` — happy path (idempotent S3 module passes
      both variants)
- [ ] Add `TestAssertIdempotent_DetectsDrift` integration test — injects
      synthetic drift via a `local-exec` provisioner that changes a non-managed
      resource between Apply and the idempotency check, asserts the check fails
- [ ] Update `docs/examples/` with a new `08-idempotency.md` example + matching
      `Test_Example08_Idempotency` in `examples_integration_test.go`
- [ ] Update `docs/examples/README.md` index
- [ ] Update `README.md` Features list to mention idempotency assertions
- [ ] Update `CLAUDE.md` Context API surface section
- [ ] Run `make ci` clean

#### Success Criteria

- Both variants compile and have doc comments
- Integration tests pass against LocalStack (`make test-integration`)
- Synthetic drift test fails the assertion as expected

---

### Phase 5: `assert/tags` package

Service-agnostic tag propagation assertion backed by the AWS Resource Groups
Tagging API (`resourcegroupstaggingapi.GetResources`).

#### Tasks

- [ ] Add `awsx/resourcegroupstaggingapi.go` with
      `NewResourceGroupsTagging(cfg aws.Config)` constructor
- [ ] Create `assert/tags/tags.go` (`package tags`) with
      `PropagatesFromRoot(tb, cfg, baseline, arns...)` and
      `PropagatesFromRootContext(tb, ctx, cfg, baseline, arns...)`
- [ ] Implement subset-check semantics: every key/value in `baseline` must be
      present on every ARN; extra tags on the resource are allowed
- [ ] Collect errors across all ARNs before calling `tb.Errorf` — surface
      "resource X is missing tag Y" + "resource X has tag Y with value Z,
      expected W" all at once
- [ ] Verify LocalStack OSS (`2026.04.0`) support for the Resource Groups
      Tagging API. Decision tree (resolved per
      [Resolved Question 4](#resolved-questions)): - **If OSS supports it:**
      ship the unit + integration path as designed - **If OSS partially supports
      it:** mock the missing endpoints in
      `sneakystack/services/resourcegroupstaggingapi/` (matches the
      `iam-identity-center`/`organizations` pattern from DESIGN-0001) - **If OSS
      does not support it:** gate `assert/tags.PropagatesFromRoot` integration
      coverage behind `libtftest.RequirePro(tb)` and document the gate in the
      package doc
- [ ] Create `assert/tags/tags_test.go` with unit tests via `internal/testfake`
      covering: missing key, wrong value, multiple-ARN aggregation, cancellation
      propagation
- [ ] Add `assert/tags` integration test in `libtftest_integration_test.go` (or
      new package-local file) using a small Terraform module that creates 2–3
      tagged resources
- [ ] Update `docs/examples/` with a new `09-tag-propagation.md` example +
      matching runnable test
- [ ] Update `docs/examples/README.md` index
- [ ] Update `README.md` Features list
- [ ] Run `make ci` clean

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

- [ ] Create `assert/snapshot/snapshot.go` (`package snapshot`) with
      `JSONStrict(tb, actual, path)` and `JSONStructural(tb, actual, path)`
- [ ] Implement structural normalization: recursively sort keys, strip
      insignificant whitespace, normalize numeric types where JSON's spec is
      ambiguous
- [ ] Wire `LIBTFTEST_UPDATE_SNAPSHOTS=1` rewrite protocol — on mismatch,
      overwrite `path` with `actual` and pass the test; log via `tb.Logf` so CI
      runs surface what was overwritten
- [ ] Implement `ExtractIAMPolicies(planJSON []byte) (map[string][]byte, error)`
      — walks `planned_values.root_module.resources` for `aws_iam_role`,
      `aws_iam_policy`, `aws_iam_role_policy`; returns one entry per role per
      policy keyed by
      `<resource_address>.<assume_role|inline:<name>|managed:<arn>>`
- [ ] Implement `ExtractResourceAttribute(planJSON, addr, path) ([]byte, error)`
      — generic JSON path extraction under
      `planned_values.root_module.resources[?address==addr].values.<path>`
- [ ] Create `assert/snapshot/snapshot_test.go` covering: identical JSON,
      byte-different-but-structurally-equal (strict fails, structural passes),
      missing snapshot file (without update mode → fail; with update mode →
      write + pass), structurally different JSON (both forms fail)
- [ ] Create `assert/snapshot/extract_test.go` covering: extract IAM policies
      from a fixture plan JSON, extract a KMS key policy via the generic helper,
      missing resource address (returns error)
- [ ] Generate fixture plan JSON for tests: small Terraform module with one IAM
      role + one KMS key, capture `terraform show -json plan.out` as
      `testdata/plan-iam-kms.json`
- [ ] Update `docs/examples/` with a new `10-snapshot-iam.md` example + matching
      runnable test
- [ ] Update `docs/examples/README.md` index
- [ ] Update `README.md` Features list
- [ ] Run `make ci` clean

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

- [ ] Create `tools/docgen/main.go` (`package main`) with a `scan` subcommand
      that: - Walks the repo's `.go` files (respecting `.gitignore`) - Pairs
      each `// libtftest:requires <tags> <reason>` line with the immediately
      following function declaration (parses with `go/parser` for declaration
      positions only — keeps the regex/AST boundary clean) - Emits a JSON
      intermediate representation (function name, package path, tags, reason,
      source location)
- [ ] Add a `render` subcommand that consumes the JSON IR and writes
      `docs/feature-matrix.md` — one row per function, one column per distinct
      tag encountered, with the reason rendered alongside
- [ ] Add a `check` subcommand that: - Walks the repo for calls to
      `libtftest.RequirePro(` (regex plus simple scope detection — the enclosing
      function) - Fails (exit non-zero, log the offending file:line) when any
      such function lacks a `// libtftest:requires` marker
- [ ] Add `make docs-matrix` target that runs `tools/docgen render`
- [ ] Add `make check-markers` target that runs `tools/docgen check`
- [ ] Wire `make check-markers` into `make ci`
- [ ] Add `tools/docgen/main_test.go` with table-driven tests covering:
      single-tag marker, multi-tag marker, missing marker (caught by `check`),
      function with marker but no `RequirePro` call (allowed — markers may
      anticipate future gates)
- [ ] Add `tools/doc.go` documenting the directory's purpose
- [ ] Run `tools/docgen render` and commit the initial `docs/feature-matrix.md`
- [ ] Update `README.md` to link to `docs/feature-matrix.md`
- [ ] Update `CLAUDE.md` to mention the `tools/docgen` location

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

#### Tasks

- [ ] Bump `plugins/libtftest/.claude-plugin/plugin.json` version 0.2.0 → 0.3.0
- [ ] Matching bump in `.claude-plugin/marketplace.json`
- [ ] Version-pin range across all `tftest:*` skill bodies, `_version-check.md`,
      `_frontmatter.md`, `README.md`, and the reviewer agent: `>=0.1.0, <1.0.0`
      → `>=0.2.0, <1.0.0`
- [ ] Update `tftest:add-test` SKILL.md + scaffold to use the new import shape
- [ ] Update `tftest:add-assertion` SKILL.md + scaffold to use the new
      per-service-package shape
- [ ] Update `tftest:add-fixture` SKILL.md + scaffold to use the new
      per-service-package shape
- [ ] Update `tftest:scaffold` (single-layout template) to use the new import
      shape; add `AssertIdempotent` mention as a module-hygiene convention
- [ ] Update umbrella `tftest` SKILL.md to surface the new module-hygiene
      primitives (idempotency, tags, snapshot)
- [ ] Update `plugins/libtftest/CHANGELOG.md` with a `[0.3.0]` entry explaining:
      the libtftest v0.5.0 (or whichever final tag) API changes the plugin
      tracks, what changed for skill consumers, and the SemVer split
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
- Separate PR ready to open in the `claude-skills` repo

---

### Phase 9: Release verification

Cross-cutting verification that runs once the PR opens and again after merge.
Not a separate PR.

#### Tasks

- [ ] PR CI green (lint, test, integration, docker, drift check, claudelint)
- [ ] PR merges to `main` with the `minor` label
- [ ] `Bump Version` + `Release` + `Changelog Sync` + `Docker` workflow jobs all
      green on the post-merge run
- [ ] `v0.2.0` tag + GitHub Release published with goreleaser notes
- [ ] Multi-arch `sneakystack` image at
      `ghcr.io/donaldgifford/sneakystack:0.2.0` signed via cosign keyless
- [ ] Plugin sync PR (Phase 8) in `donaldgifford/claude-skills` lands; plugin
      v0.3.0 published; pin range covers libtftest `>=0.2.0, <1.0.0`
- [ ] No `chore(deps)` dependabot PRs left orphaned
- [ ] `CHANGELOG.md` on `main` reflects the v0.2.0 section produced by
      `git-cliff` without any manual fixups
- [ ] Update memory `MEMORY.md` pointer to a new memory entry summarizing the
      layout-change shape (deferred to post-merge)

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

- [ ] Unit test: each new `assert/<service>` package mirrors the coverage the
      pre-refactor file had — at minimum the `*Context_PropagatesCancel` test
      per assertion
- [ ] Unit test: each new `fixtures/<service>` package covers the cancellation +
      `WithoutCancel` cleanup pattern from INV-0001
- [ ] Unit test: `assert/tags` covers missing-key, wrong-value, multi-ARN
      aggregation, ctx propagation
- [ ] Unit test: `assert/snapshot` covers identical / structurally- equal /
      different / missing-file / update-mode scenarios for both strict and
      structural variants
- [ ] Unit test: `assert/snapshot.ExtractIAMPolicies` against fixture plan JSON
      containing `aws_iam_role` + `aws_iam_policy`
- [ ] Unit test: `assert/snapshot.ExtractResourceAttribute` against a non-IAM
      resource type (KMS key)
- [ ] Integration test: end-to-end `TestCase.AssertIdempotent` against a
      known-idempotent S3 module
- [ ] Integration test: end-to-end `TestCase.AssertIdempotentApply` same module
      — succeeds despite the extra Apply round-trip
- [ ] Integration test: synthetic drift causes `AssertIdempotent` to fail
- [ ] Integration test: `assert/tags.PropagatesFromRoot` against a module that
      tags 2–3 resources with a known baseline
- [ ] `make ci` green on every PR
- [ ] `make test-coverage` shows no coverage regression
- [ ] `make test-examples` (Docker required) — every new `examples/0N-*.md` has
      a green matching test
- [ ] CHANGELOG drift check green on every PR

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
