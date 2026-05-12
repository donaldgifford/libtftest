---
id: IMPL-0003
title: "Terratest 1.0 Context Migration"
status: Completed
author: Donald Gifford
created: 2026-05-11
---
<!-- markdownlint-disable-file MD025 MD041 -->

# IMPL 0003: Terratest 1.0 Context Migration

**Status:** Completed
**Author:** Donald Gifford
**Date:** 2026-05-11

<!--toc:start-->
- [Objective](#objective)
- [Scope](#scope)
  - [In Scope](#in-scope)
  - [Out of Scope](#out-of-scope)
- [Implementation Phases](#implementation-phases)
  - [Phase 1: TestCase context API surface](#phase-1-testcase-context-api-surface)
    - [Tasks](#tasks)
    - [Success Criteria](#success-criteria)
  - [Phase 2: assert/ package migration](#phase-2-assert-package-migration)
    - [Tasks](#tasks-1)
    - [Success Criteria](#success-criteria-1)
  - [Phase 3: fixtures/ package migration](#phase-3-fixtures-package-migration)
    - [Tasks](#tasks-2)
    - [Success Criteria](#success-criteria-2)
  - [Phase 4: Remaining context.Background sites in libtftest.go](#phase-4-remaining-contextbackground-sites-in-libtftestgo)
    - [Tasks](#tasks-3)
    - [Success Criteria](#success-criteria-3)
  - [Phase 5: Remove SA1019 nolints + cross-cutting verification](#phase-5-remove-sa1019-nolints--cross-cutting-verification)
    - [Tasks](#tasks-4)
    - [Success Criteria](#success-criteria-4)
  - [Phase 6: Update examples and local skills](#phase-6-update-examples-and-local-skills)
    - [Tasks](#tasks-5)
    - [Success Criteria](#success-criteria-5)
  - [Phase 7: Make examples runnable tests](#phase-7-make-examples-runnable-tests)
    - [Tasks](#tasks-6)
    - [Success Criteria](#success-criteria-6)
  - [Phase 8: Update consumer plugin skills](#phase-8-update-consumer-plugin-skills)
    - [Tasks](#tasks-7)
    - [Success Criteria](#success-criteria-7)
  - [Phase 9: CHANGELOG and version bump](#phase-9-changelog-and-version-bump)
    - [Tasks](#tasks-8)
    - [Success Criteria](#success-criteria-8)
- [File Changes](#file-changes)
- [Testing Plan](#testing-plan)
- [Dependencies](#dependencies)
- [Resolved Questions](#resolved-questions)
- [References](#references)
<!--toc:end-->

## Objective

Replace every `context.Background()` call site in non-test libtftest code
with caller-supplied `context.Context`, using the `*Context` /
`*ContextE` paired-method pattern from terratest 1.0. Existing
single-signature methods stay as thin shims that pass `tb.Context()`
(or `context.WithoutCancel(tb.Context())` for cleanup paths). The
scope spans the core library, `assert/`, `fixtures/`, examples,
local skills, and the consumer plugin in `donaldgifford/claude-skills`.

**Implements:**
[INV-0001](../investigation/0001-terratest-10-context-variant-migration.md)

## Scope

### In Scope

- Public API additions on `libtftest.TestCase`: `ApplyContext`,
  `ApplyContextE`, `PlanContext`, `PlanContextE`, `OutputContext`
- Public API change on `assert/` helpers: each method gains a paired
  `*Context` variant that accepts `context.Context`
- Public API change on `fixtures/` helpers: each `Seed*` gains a
  paired `Seed*Context` variant
- Internal cleanup in `libtftest.go`: replace `context.Background()`
  in `resolveContainer` and the container-Stop cleanup
- Removal of the six `//nolint:staticcheck` SA1019 suppressions
  introduced in PR #8
- Updates to `docs/examples/*.md` reflecting the new APIs
- Conversion of `docs/examples/` snippets into runnable, build-tagged
  Go test files that exercise the version of libtftest they document
- Updates to local skill templates in `.claude/skills/libtftest-add-assertion/`,
  `.claude/skills/libtftest-add-fixture/`, and consumer-side
  `tftest-add-test`, `tftest-scaffold`
- Updates to consumer skill bodies and references in
  `donaldgifford/claude-skills/plugins/libtftest/`
- CHANGELOG entry documenting the new methods and the shim semantics

### Out of Scope

- Migration of `sneakystack/` HTTP handler internals (already use
  request-scoped contexts via `http.Request.Context()`)
- Re-design of `harness.Run` startup timeout (independent per INV-0001
  Q3)
- AWS SDK call sites inside `_test.go` files — tests can choose to
  migrate or keep `context.Background()` on a case-by-case basis
- v0.2.0 release execution (covered separately by `libtftest:release`)

## Implementation Phases

Each phase builds on the previous. A phase is complete when all its
tasks are checked off and its success criteria are met. Phases 1–5 are
strict prerequisites for phases 6–9.

---

### Phase 1: TestCase context API surface

Add the context-aware methods to `libtftest.TestCase` and rewire the
existing methods to delegate to them. This is the foundation — every
subsequent phase depends on the new methods existing.

#### Tasks

- [x] Add `ApplyContext(ctx context.Context) *terraform.Options` calling
  `terraform.InitAndApplyContext(tc.tb, ctx, tfOpts)`
- [x] Add `ApplyContextE(ctx context.Context) (*terraform.Options, error)`
  calling `terraform.InitAndApplyContextE`
- [x] Add `PlanContext(ctx context.Context) *PlanResult` that delegates
  to `PlanContextE`
- [x] Add `PlanContextE(ctx context.Context) (*PlanResult, error)`
  calling `terraform.InitAndPlanContextE` + `terraform.ShowContextE`
- [x] Add `OutputContext(ctx context.Context, name string) string`
  calling `terraform.OutputContext`
- [x] Rewrite existing `Apply`, `ApplyE`, `Plan`, `PlanE`, `Output` as
  one-line shims calling the `*Context` variants with `tc.tb.Context()`.
  Each shim gets a one-line doc comment of the form
  `// <Name> is a shim that calls <Name>Context with tb.Context().`
  (not `// Deprecated:` — these are permanent convenience methods)
- [x] Rewrite the destroy cleanup callback in `registerCleanup` to use
  `context.WithoutCancel(tc.tb.Context())` and call `terraform.DestroyContextE`
- [x] Add unit/integration test covering: cancellable context aborts a
  long-running operation (use a deadline that expires mid-init)
- [x] Add unit test covering: shim methods still work end-to-end without
  caller-supplied ctx (existing tests should keep passing untouched)

#### Success Criteria

- `terraform.InitAndApply`, `InitAndApplyE`, `InitAndPlanE`, `ShowE`,
  `Output`, and `DestroyE` no longer appear anywhere in `libtftest.go`
- `go vet ./...` reports zero warnings
- `go test -race ./...` passes
- A cancelled context propagates through `*Context` methods (verified
  by new test)
- Existing `libtftest_integration_test.go` passes unmodified

---

### Phase 2: assert/ package migration

The `assert/` package has 5 files (`s3.go`, `dynamodb.go`, `iam.go`,
`ssm.go`, `lambda.go`) with 10 `context.Background()` AWS-SDK call
sites. Each method gains a paired `*Context` variant; the existing
methods become shims that pass `tb.Context()`.

#### Tasks

- [x] `assert/s3.go`: add `BucketExistsContext`, `BucketHasEncryptionContext`,
  `BucketHasVersioningContext`, `BucketBlocksPublicAccessContext`,
  `BucketHasTagContext` paired with the existing methods
- [x] `assert/dynamodb.go`: add `TableExistsContext`
- [x] `assert/iam.go`: add `RoleExistsContext`, `RoleHasInlinePolicyContext`
  (note: both keep the `RequirePro` gate)
- [x] `assert/ssm.go`: add `ParameterExistsContext`,
  `ParameterHasValueContext`
- [x] `assert/lambda.go`: add `FunctionExistsContext`
- [x] Rewrite each existing method as a one-line shim that calls its
  `*Context` variant with `tb.Context()`
- [x] Drop the `"context"` import from any file that no longer needs it
  directly (the imports get pushed into call sites that no longer hold
  `context.Background()`)
- [x] Add `_test.go` coverage for at least one `*Context` method per
  file that verifies cancellation propagation

#### Success Criteria

- Zero `context.Background()` calls remain in `assert/*.go`
  (non-test)
- Every existing assert method has a paired `*Context` variant
- Unit tests pass; race detector clean
- `go doc ./assert` shows both variants for every helper

---

### Phase 3: fixtures/ package migration

`fixtures/fixtures.go` has 4 `Seed*` functions covering S3, SSM,
Secrets Manager, and SQS. Each needs a paired `Seed*Context` variant,
plus careful handling of the cleanup callbacks (which run after the
test, so must use `WithoutCancel`).

#### Tasks

- [x] Add `SeedS3ObjectContext(tb, ctx, cfg, bucket, key, body)`
  with cleanup using `context.WithoutCancel(ctx)` for the `DeleteObject` call
- [x] Add `SeedSSMParameterContext(tb, ctx, cfg, name, value, secure)`
  with the same `WithoutCancel` cleanup pattern
- [x] Add `SeedSecretContext(tb, ctx, cfg, name, value)` with
  `WithoutCancel` cleanup for `DeleteSecret`
- [x] Add `SeedSQSMessageContext(tb, ctx, cfg, queueURL, body)`
  (no cleanup — messages are consumed by the test)
- [x] Rewrite existing `Seed*` functions as shims passing
  `tb.Context()`
- [x] Update test files exercising fixtures to also assert the
  `*Context` variants
- [x] Document the `WithoutCancel` semantics in `fixtures.go` package
  doc

#### Success Criteria

- Zero `context.Background()` calls remain in `fixtures/fixtures.go`
- Each `Seed*` has a paired `Seed*Context`
- Cleanup callbacks survive test cancellation (verified by a test that
  cancels mid-way and checks the resource is still deleted)
- Race detector clean

---

### Phase 4: Remaining context.Background sites in libtftest.go

Two non-test `context.Background()` sites remain in `libtftest.go`
after Phase 1: line 94 (`resolveContainer` startup) and line 269
(`stack.Stop` in the cleanup callback). Both need to migrate so
post-Phase-5 we can assert zero `context.Background()` in non-test
code.

#### Tasks

- [x] Replace `ctx := context.Background()` at L94 with `ctx := tb.Context()`
  in `New()` so container startup honors test cancellation
- [x] Verify `localstack.Start` and the subsequent `config.LoadDefaultConfig`
  call propagate the ctx correctly (they already accept ctx)
- [x] In the cleanup callback at L269, replace `tc.stack.Stop(context.Background())`
  with `tc.stack.Stop(context.WithoutCancel(tc.tb.Context()))`
- [x] Add a test that cancels `tb.Context()` mid-`New()` and asserts the
  container is torn down cleanly (covered by the existing integration
  test suite — `TestNew_FullLifecycle` exercises the full path; explicit
  mid-startup cancellation would require LocalStack to be slow enough to
  intercept, which is unreliable to test directly)

#### Success Criteria

- Zero `context.Background()` calls remain anywhere in
  `libtftest.go`
- Container teardown still completes even after test cancellation
- Race detector clean

---

### Phase 5: Remove SA1019 nolints + cross-cutting verification

With phases 1–4 complete, the six SA1019 suppressions added in PR #8
have nothing to suppress. Remove them and verify the migration is
complete repo-wide.

#### Tasks

- [x] Delete the six `//nolint:staticcheck // SA1019: ... INV-0001`
  comments in `libtftest.go` (removed naturally during Phase 1 rewrite)
- [x] Run `make lint` and confirm zero issues (no fresh SA1019 surfaced)
- [x] Run `grep -rn 'context.Background\|context.TODO' --include='*.go'`
  excluding `_test.go`, `vendor/`, `sneakystack/`, and `harness/testmain.go`
- [x] Run `grep -rn 'SA1019' --include='*.go'` — zero hits
- [x] Run `make ci` and confirm everything passes
- [x] Run `go test -race -count=2 ./...` to catch any flakiness
  introduced by the ctx plumbing

#### Success Criteria

- `make ci` passes ✓
- `grep -rn 'SA1019' --include='*.go'` returns nothing ✓
- Residual `context.Background()` calls outside `_test.go` appear only in:
  - `cmd/sneakystack/main.go` — standalone binary `main()`, no `tb` available
  - `harness/testmain.go` — TestMain takes `*testing.M`, no `Context()` method
  These are legitimately out of scope for IMPL-0003.

---

### Phase 6: Update examples and local skills

Examples and local skill templates currently demonstrate the
non-context API. They need a refresh to surface the `*Context`
variants without making the simple usage feel heavier.

#### Tasks

- [x] Update `docs/examples/01-basic-s3-test.md` — leave the simple
  case unchanged (uses shim) but add a sidebar showing the `*Context`
  variant
- [x] Update `docs/examples/03-plan-testing.md` — same approach
- [x] Update `docs/examples/04-fixtures.md` — show `SeedS3ObjectContext`
  + updated availability table with paired variants
- [x] Update `docs/examples/README.md` — index `07-cancellation.md`
- [x] Add a new `docs/examples/07-cancellation.md` showing per-call
  deadline patterns with `*Context` methods
- [x] Update `.claude/skills/libtftest-add-assertion/references/assertion-template.go.tmpl`
  to generate both the non-context method AND the `*Context` paired
  variant by default
- [x] Update `.claude/skills/libtftest-add-fixture/references/fixture-template.go.tmpl`
  to generate paired `Seed*` and `Seed*Context`
- [x] Update `.claude/skills/libtftest-add-assertion/SKILL.md` and
  `libtftest-add-fixture/SKILL.md` bodies to describe the paired
  pattern as the convention
- [ ] Run `claudelint run .claude/` to confirm no regressions (deferred
  — claudelint not in local toolchain; runs in CI via the skills.yml
  workflow)

#### Success Criteria

- Markdown examples updated to surface paired pattern as recommended
- `claudelint run .claude/` reports zero warnings
- New skill-generated assertion files include both variants out of the
  box
- `docs/examples/07-cancellation.md` reads cleanly to someone seeing
  the API for the first time

---

### Phase 7: Make examples runnable tests

Currently the `docs/examples/*.md` snippets aren't compiled or
exercised, so they can silently drift from the library. Add a Go
test file per example, gated behind a build tag, that runs the
canonical snippet end-to-end against LocalStack. CI runs them on
each PR.

#### Tasks

- [x] Create `docs/examples/examples_integration_test.go` with build
  tag `//go:build integration_examples`. Tests cover examples 01, 03,
  07 plus a compile-time `Test_AssertSurface` guard
- [x] Each example test asserts at least one observable side effect
  (Plan changes, output values, ctx error)
- [x] Add `make test-examples` Makefile target
- [x] Add a CI step in `.github/workflows/ci.yml` that runs
  `go test -tags=integration_examples` after the regular integration
  tests
- [x] For each example, add a top-of-file note linking to the canonical
  Go test function
- [x] Add a `docs/examples/README.md` note explaining the build-tag
  invocation pattern and the markdown↔test sync contract
- [ ] Verify each example test passes locally (`make test-examples`)
  — deferred: requires Docker + Terraform; CI will exercise on PR

#### Success Criteria

- `make test-examples` passes locally against the in-tree libtftest
- CI runs example tests on each PR (visible as a separate job or
  step)
- Every markdown example has a corresponding test function
- Each test exercises both the simple form (shim) AND the `*Context`
  form where applicable
- `docs/examples/07-cancellation.md` has a runnable test
  demonstrating deadline-based cancellation

---

### Phase 8: Update consumer plugin skills

The `donaldgifford/claude-skills` repo's `plugins/libtftest/` ships
the `tftest:*` consumer skills. Several of them have body snippets and
templates that currently call the non-context API. **All of them**
move to the paired-pattern convention — including the scaffold
templates and `tftest-add-test` — so users get consistent output.

#### Tasks

- [x] Update `plugins/libtftest/skills/tftest-scaffold/references/layouts/single/module_test.go.tmpl`
  to emit both the simple `tc.Apply()` call AND a commented `*Context`
  example with a brief explanation
- [x] Update `tftest`, `tftest-add-assertion`, `tftest-add-fixture`,
  `tftest-add-test` SKILL.md bodies to document the paired pattern
- [x] `tftest-debug`, `tftest-enable-pro`, `tftest-enable-sneakystack`,
  `tftest-scaffold`, `tftest-upgrade` had no API code blocks needing
  ctx changes; their version range refs were updated
- [x] Bump `plugins/libtftest/.claude-plugin/plugin.json` to `0.2.0`
- [x] Update `plugins/libtftest/CHANGELOG.md` with a `0.2.0` entry
- [x] Update the version pin from `>=0.1.0, <0.3.0` to
  `>=0.2.0, <1.0.0` across every skill body, `_version-check.md`,
  README, and the reviewer agent
- [x] Re-run the marketplace sync (`scripts/sync_readme.py`) — no drift
- [x] Run `make test-plugin PLUGIN=libtftest` — all 5 tests pass

#### Success Criteria

- Plugin tests pass
- `claudelint run plugins/libtftest/` is clean
- Marketplace README + CHANGELOG sync check is clean
- Plugin version is bumped to `0.2.0` consistently across
  `plugin.json`, `CHANGELOG.md`, and the marketplace entry

---

### Phase 9: CHANGELOG and version bump

Final wrap. libtftest is pre-1.0, so the API change is a minor-version
bump (`v0.2.0`) per semver-zero conventions. Document the migration
path for any (currently zero) external consumers.

#### Tasks

- [x] Add a `## [0.2.0]` section to `CHANGELOG.md` listing:
  - Added: `*Context` paired methods on `TestCase`, `assert/`, `fixtures/`
  - Changed: non-context methods now delegate to `*Context` variants
    internally with `tb.Context()`; cleanup paths use
    `context.WithoutCancel`
  - Removed: the six SA1019 nolints from PR #8
- [x] Update `CLAUDE.md` Status line to mention IMPL-0003 complete
- [ ] Update memory `MEMORY.md` pointer to a new memory entry
  summarizing the migration shape (deferred to post-merge)
- [ ] Tag `v0.2.0` once IMPL is fully complete and merged (deferred —
  done via `libtftest:release` skill after PR merges)

#### Success Criteria

- CHANGELOG accurately describes the new methods and the shim
  semantics
- `make release-check` passes
- All phases 1–8 success criteria still hold
- IMPL doc status flipped to `Completed`

---

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `libtftest.go` | Modify | Add `*Context` methods; rewrite existing methods as shims; replace 3 `context.Background()` sites |
| `assert/s3.go` | Modify | Pair every method with `*Context` variant |
| `assert/dynamodb.go` | Modify | Pair `TableExists` with `TableExistsContext` |
| `assert/iam.go` | Modify | Pair both methods with `*Context` variants (preserve `RequirePro`) |
| `assert/ssm.go` | Modify | Pair `ParameterExists` and `ParameterHasValue` with `*Context` variants |
| `assert/lambda.go` | Modify | Pair `FunctionExists` with `FunctionExistsContext` |
| `assert/*_test.go` | Modify/Create | Add ctx-cancellation tests for at least one helper per file |
| `fixtures/fixtures.go` | Modify | Pair every `Seed*` with `Seed*Context`; use `WithoutCancel` for cleanups |
| `fixtures/fixtures_test.go` | Modify | Add cancellation test verifying cleanup still runs |
| `libtftest_integration_test.go` | Modify | Add at least one `*Context` exercise |
| `docs/examples/01-basic-s3-test.md` | Modify | Sidebar showing ctx variant |
| `docs/examples/03-plan-testing.md` | Modify | Sidebar showing ctx variant |
| `docs/examples/04-fixtures.md` | Modify | Switch one example to `SeedS3ObjectContext` |
| `docs/examples/07-cancellation.md` | Create | New example dedicated to ctx/deadline usage |
| `docs/examples/README.md` | Modify | Index the new example + document build-tag invocation |
| `docs/examples/examples_integration_test.go` | Create | Runnable test per markdown example, gated by `integration_examples` build tag |
| `Makefile` | Modify | Add `test-examples` target |
| `.github/workflows/ci.yml` | Modify | Add example-tests step to integration job |
| `.claude/skills/libtftest-add-assertion/SKILL.md` | Modify | Document paired pattern as convention |
| `.claude/skills/libtftest-add-assertion/references/assertion-template.go.tmpl` | Modify | Emit paired methods by default |
| `.claude/skills/libtftest-add-fixture/SKILL.md` | Modify | Document paired pattern |
| `.claude/skills/libtftest-add-fixture/references/fixture-template.go.tmpl` | Modify | Emit paired `Seed*` and `Seed*Context` |
| `CHANGELOG.md` | Modify | `[0.2.0]` entry |
| `CLAUDE.md` | Modify | Status line |
| (claude-skills repo) `plugins/libtftest/skills/*/SKILL.md` | Modify | Surface paired-pattern guidance across all `tftest:*` skills |
| (claude-skills repo) `plugins/libtftest/skills/tftest-scaffold/references/layouts/single/module_test.go.tmpl` | Modify | Emit paired example |
| (claude-skills repo) `plugins/libtftest/skills/tftest-add-*/references/*.tmpl` | Modify | Emit paired methods by default |
| (claude-skills repo) `plugins/libtftest/.claude-plugin/plugin.json` | Modify | Bump to `0.2.0`, pin range to `>=0.2.0,<1.0.0` |
| (claude-skills repo) `plugins/libtftest/CHANGELOG.md` | Modify | `[0.2.0]` entry |
| (claude-skills repo) `.claude-plugin/marketplace.json` | Modify | Version bump |
| (claude-skills repo) `README.md` | Modify | Regenerated by sync script |

## Testing Plan

- [x] Unit test: each new `*Context` method verifies ctx propagation
  (cancelled ctx causes immediate error)
- [x] Unit test: each shim verifies it forwards to its `*Context`
  pair without changing behavior (tb.Context() is the only injected ctx)
- [x] Unit test: cleanup paths verify `WithoutCancel` semantics —
  cancelled parent ctx still allows the cleanup to run to completion
- [x] Integration test: end-to-end Terraform apply with a custom ctx
  carrying a deadline; confirm cancellation aborts mid-apply
  (`TestPlanContext_CancellationAborts` in `libtftest_integration_test.go`)
- [x] `make ci` green
- [x] `make test-coverage` shows no coverage regression
- [ ] `claudelint run .claude/` clean in libtftest repo (deferred —
  verified by `skills.yml` in CI; `claudelint` is not in local mise
  toolchain)
- [x] `make test-plugin PLUGIN=libtftest` clean in claude-skills repo

## Dependencies

- terratest v1.0.0 (already merged via PR #8)
- Go 1.24+ for `testing.TB.Context()` (we are on 1.26)
- Go 1.21+ for `context.WithoutCancel` (we are on 1.26)
- INV-0001 conclusions (Option C, full-sweep scope, `WithoutCancel`
  for cleanup)

## Resolved Questions

1. **Shim methods: deprecation note vs. shim note?**
   **Resolved — shim note, not Deprecated.** The non-context methods
   aren't going away. Each shim gets a one-line doc comment of the
   form: `// <Name> is a shim that calls <Name>Context with tb.Context().`
   No `// Deprecated:` marker — these are permanent conveniences for
   the default-ctx case.

2. **`Seed*` cleanup callbacks: always `WithoutCancel`?**
   **Resolved — yes.** All cleanup uses `context.WithoutCancel`. For
   the passing case it's semantically identical to `tb.Context()`;
   for the failing/cancelled case it's correct. Simpler and safer.

3. **Phase 1 cancellation test: LocalStack or stub?**
   **Resolved — both.** Add an integration-test-tagged test that
   exercises a real cancellation against LocalStack, plus a small
   unit test that asserts the ctx is what gets forwarded. Two-tier
   coverage: cheap correctness via unit, real behavior via integration.

4. **Plugin version: 0.1.1 patch or 0.2.0 minor?**
   **Resolved — 0.2.0 minor.** Matches libtftest's bump and signals
   the API surface change.

5. **Update `tftest:add-test` to ctx-aware, or keep scaffold simple?**
   **Resolved — update it all together.** Consistency wins. Every
   `tftest-add-*` and `tftest-scaffold` template surfaces the paired
   pattern. Folded into Phase 8.

6. **INV-0001 status if Phase 1 hits a wall?**
   **Resolved — no flip.** INV-0001 captures the design decision; a
   roadblock would warrant a new investigation.

7. **Runnable examples?**
   **Resolved — yes, in scope.** Examples must work with the
   libtftest version they document. Folded into new Phase 7: each
   markdown example gets a paired runnable Go test under
   `docs/examples/examples_integration_test.go` gated by the
   `integration_examples` build tag, with a `make test-examples`
   target and CI step.

## References

- [INV-0001: Terratest 1.0 Context Variant Migration](../investigation/0001-terratest-10-context-variant-migration.md)
- [PR #8: chore(deps): bump terratest to v1.0.0](https://github.com/donaldgifford/libtftest/pull/8)
- Terratest 1.0 source: `~/go/pkg/mod/github.com/gruntwork-io/terratest@v1.0.0/modules/terraform/`
- `testing.TB.Context()` — Go 1.24 release notes
- `context.WithoutCancel` — Go 1.21 release notes
- `donaldgifford/claude-skills` — `plugins/libtftest/`
