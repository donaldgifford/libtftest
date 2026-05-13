---
id: DESIGN-0002
title: "Claude Skills for libtftest authors and consumers"
status: Draft
author: Donald Gifford
created: 2026-04-29
---

<!-- markdownlint-disable-file MD025 MD041 -->

# DESIGN 0002: Claude Skills for libtftest authors and consumers

**Status:** Draft **Author:** Donald Gifford **Date:** 2026-04-29

<!--toc:start-->
- [Overview](#overview)
- [Goals and Non-Goals](#goals-and-non-goals)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Background](#background)
  - [What is a Claude skill?](#what-is-a-claude-skill)
  - [Two audiences, two homes](#two-audiences-two-homes)
- [Detailed Design](#detailed-design)
  - [Audience 1: libtftest authors (local .claude/skills/)](#audience-1-libtftest-authors-local-claudeskills)
    - [libtftest:add-assertion](#libtftestadd-assertion)
    - [libtftest:add-fixture](#libtftestadd-fixture)
    - [libtftest:add-awsx-client](#libtftestadd-awsx-client)
    - [libtftest:add-sneakystack-service](#libtftestadd-sneakystack-service)
    - [libtftest:add-localstack-default](#libtftestadd-localstack-default)
    - [libtftest:bump-localstack](#libtftestbump-localstack)
    - [libtftest:release](#libtftestrelease)
    - [libtftest:update-design (agent)](#libtftestupdate-design-agent)
    - [libtftest-reviewer (agent)](#libtftest-reviewer-agent)
  - [Audience 2: libtftest consumers (plugin skills in donaldgifford/claude-skills)](#audience-2-libtftest-consumers-plugin-skills-in-donaldgiffordclaude-skills)
    - [tftest](#tftest)
    - [tftest:scaffold](#tftestscaffold)
    - [tftest:add-test](#tftestadd-test)
    - [tftest:add-fixture](#tftestadd-fixture)
    - [tftest:add-assertion](#tftestadd-assertion)
    - [tftest:setup-ci](#tftestsetup-ci)
    - [tftest:enable-pro](#tftestenable-pro)
    - [tftest:enable-sneakystack](#tftestenable-sneakystack)
    - [tftest:debug](#tftestdebug)
    - [tftest:upgrade](#tftestupgrade)
    - [tftest-reviewer (agent)](#tftest-reviewer-agent)
- [API / Interface Changes](#api--interface-changes)
- [Data Model](#data-model)
- [Testing Strategy](#testing-strategy)
- [Migration / Rollout Plan](#migration--rollout-plan)
- [Resolved Decisions](#resolved-decisions)
- [Open Questions](#open-questions)
- [References](#references)
<!--toc:end-->

## Overview

This design enumerates the Claude Code skills and agents we should build to make
libtftest fast and consistent to extend (for maintainers of this repo) and easy
to adopt (for Terraform module authors who consume libtftest in their own
repos). Skills are split into two groups by audience: **local skills** that live
inside this repo and accelerate libtftest development itself, and **plugin
skills** that ship in a dedicated `libtftest` plugin under
`donaldgifford/claude-skills` and help downstream consumers.

## Goals and Non-Goals

### Goals

- Codify the repeatable scaffolding patterns in libtftest (assertions, fixtures,
  awsx clients, sneakystack handlers) so adding a new one takes minutes and
  matches existing conventions.
- Make libtftest adoption in a Terraform module repo a single skill invocation
  away — no copy/paste from another repo, no hunting through docs.
- Encode our hard-won gotchas (PortEndpoint vs Endpoint, `tofu` vs `terraform`,
  `aws.Config` hugeParam threshold, `tb` vs `t` naming) inside skills so future
  work doesn't rediscover them.
- Provide review agents that can run independently from the implementing model
  and catch mistakes before PR review.
- Keep the boundary between "developing libtftest" and "using libtftest" clean
  so authors and consumers never see each other's skills.

### Non-Goals

- We do not aim to replace human review or hide complexity. Skills generate
  starting points; humans still own correctness and design.
- We do not aim to support frameworks other than libtftest in the consumer
  skills (no terratest-direct, no kitchen-terraform). A separate `tftest`
  variant could be introduced later.
- We do not maintain skills for module authors who don't want Go tests
  (`tflint`, `checkov`, `tfsec` skills are out of scope here — they're separate
  IaC plugin concerns).
- No attempt to auto-detect Pro vs Community at skill scaffolding time —
  consumers configure that explicitly.

## Background

### What is a Claude skill?

A Claude Code skill is a markdown file (`SKILL.md`) plus optional supporting
files in a directory under `.claude/skills/<name>/` (local) or under a plugin's
`skills/` directory (distributed). The frontmatter declares `name`,
`description`, and `tools`; the body contains a system prompt that's injected
when the skill activates. Skills are the primary surface for codifying
repeatable, opinionated workflows. Agents (under `.claude/agents/` or in a
plugin's `agents/`) are similar but run with a fresh context window — useful for
review, audit, or debugging tasks that benefit from a clean read.

### Two audiences, two homes

| Audience              | Where they work                       | Where their skills live                                                              |
| --------------------- | ------------------------------------- | ------------------------------------------------------------------------------------ |
| libtftest maintainers | This repo (`donaldgifford/libtftest`) | `.claude/skills/` and `.claude/agents/` in this repo (committed, team-shared)        |
| libtftest consumers   | Their own Terraform module repos      | Dedicated `libtftest` plugin in `donaldgifford/claude-skills` (installed per-machine) |

The consumer skills ship as a **dedicated `libtftest` plugin** rather than
being folded into the existing `infrastructure-as-code` plugin. Reasoning:
isolating them lets us iterate fast without breaking unrelated IaC users, and
we can merge into the IaC plugin later once the surface stabilizes.

Local skills do not need to be reusable across repos and can hardcode paths,
package names, and conventions specific to libtftest. Plugin skills must work
across many module repos with varying layouts, so they're more general and
prompt for context they need.

DESIGN-0001 §"Claude Code Automation" listed an initial cut of these skills.
This doc expands that list, adds pros/cons per skill, and adds skills not
previously considered (release automation, LocalStack version bumps, debug
helpers, upgrade helpers, review agents on both sides).

## Detailed Design

### Audience 1: libtftest authors (local `.claude/skills/`)

These ship in `.claude/skills/` in this repo. They share a common system prompt
preamble: "follow Uber Go Style Guide, stdlib-first, table-driven tests, `tb`
not `t` for `testing.TB`, comments on exported symbols end with periods, line
length 150."

#### `libtftest:add-assertion`

**Trigger:** "I want to add an assertion for `<service>`" or invoked manually.

**What it does:**

- Prompts for service name (e.g., `kms`, `cloudwatch`, `sqs`) and a list of
  assertion methods (e.g., `KeyExists`, `KeyHasPolicy`).
- Generates `assert/<service>.go` with the zero-size struct namespace pattern
  used in `assert/s3.go` and the package-level `<Service>` var.
- Generates `assert/<service>_test.go` with table-driven stubs.
- Adds the typed AWS SDK client constructor to `awsx/clients.go` if it doesn't
  exist (delegates to `libtftest:add-awsx-client`).
- Reminds the author whether the service is Community or Pro-only and inserts
  `RequirePro(t)` calls where appropriate.

**Pros:**

- Single biggest source of churn in libtftest will be assertion coverage; this
  is the highest-leverage skill.
- Keeps the namespace pattern consistent (very easy to forget the package-level
  var or the struct receiver pattern).

**Cons:**

- Pro-vs-Community gating depends on judgment that's hard to encode — skill
  needs a lookup table or human prompt.
- Coupled to `awsx` client generation; risks a sprawling skill if not careful.

#### `libtftest:add-fixture`

**Trigger:** "Add a fixture for seeding `<service>` data."

**What it does:**

- Prompts for service and a `Seed*` function name plus signature (e.g.,
  `SeedDynamoDBItem(tb, cfg, table, item)`).
- Adds the function to `fixtures/fixtures.go` with the matching `t.Cleanup`
  teardown.
- Generates a test that actually seeds against a stub or LocalStack and checks
  cleanup runs.

**Pros:**

- Cleanup pairing is the part that's most often forgotten — codifying it
  prevents the most likely bug class.
- The `tb testing.TB` naming convention (thelper linter) is easy to mess up; the
  skill enforces it.

**Cons:**

- Fixture variety is wide (idempotent seeds, cross-service seeds, multi-step
  seeds); a one-shot template may not fit all cases. The skill should generate a
  starting point, not a finished fixture.

#### `libtftest:add-awsx-client`

**Trigger:** "Add an awsx client for `<service>`."

**What it does:**

- Prompts for service name and SDK module path.
- Adds a typed constructor to `awsx/clients.go` matching existing style (returns
  `*<service>.Client`, takes `aws.Config`, applies the `BaseEndpoint`-aware
  override pattern).
- Adds it to `go.mod` indirect deps if not present.
- Generates a `_test.go` smoke test.

**Pros:**

- AWS SDK v2 client constructors are mechanical; `config.WithBaseEndpoint` vs
  deprecated `EndpointResolverV2` is exactly the kind of decision a skill should
  encode once.
- Saves having to remember the exact import path each time.

**Cons:**

- Trivial enough that `add-assertion` could absorb it; standalone existence is
  borderline.

#### `libtftest:add-sneakystack-service`

**Trigger:** "Add a sneakystack handler for `<service>`."

**What it does:**

- Scaffolds `sneakystack/services/<service>.go` with the Store-typed wrapper
  pattern (Put/Get/List/Delete typed for the service's resources).
- Registers the handler in the proxy router (`sneakystack/proxy.go`) with the
  correct `X-Amz-Target` prefix or path matcher.
- Generates a handler test using `httptest`.
- Adds an example to `docs/examples/06-sneakystack.md` showing the new service
  in the `Services` list.

**Pros:**

- sneakystack will grow service-by-service; a skill keeps the routing,
  Store-typing, and test wiring consistent.
- The proxy registration step is non-obvious and easy to forget — codifying it
  is high value.

**Cons:**

- Different AWS services use very different protocols (JSON-1.1, REST-XML,
  query) — the skill needs branches for at least the common dispatch styles.
- Likely the most complex skill on this list; may merit being two skills
  (`add-sneakystack-service-jsonrpc`, `add-sneakystack-service-restxml`).

#### `libtftest:add-localstack-default`

**Trigger:** "Enable `<service>` in the default LocalStack service list."

**What it does:**

- Updates `localstack/container.go` defaults.
- Adds the service to the README/docs default-services list.
- Reminds the author whether the service is Community or Pro and updates the
  health-endpoint matcher accordingly.

**Pros:**

- Keeps `defaults` in sync with docs.

**Cons:**

- Tiny — could be a doc-line edit. Borderline whether it needs a skill at all.
  Probably a "nice to have" not a "must have."

#### `libtftest:bump-localstack`

**Trigger:** "Bump LocalStack to `<version>`."

**What it does:**

- Updates the pinned image in `localstack/container.go` and `Dockerfile.*`.
- Updates `docs/examples/05-custom-image.md` and README references.
- Runs the integration test suite, captures any regressions, and writes a
  changelog entry under `docs/`.
- Reminds the author to check the LocalStack release notes for service removals
  or signature changes (e.g., the S3 MalformedXML compat issue we hit on 4.4).

**Pros:**

- LocalStack version bumps are infrequent but high-blast-radius; we always end
  up writing a changelog and re-running everything. Codifying the checklist
  prevents skipping steps.

**Cons:**

- Most of this is shell + docs, not generation. May be better as a Makefile
  target than a skill.

#### `libtftest:release`

**Trigger:** "Tag a release `vX.Y.Z`."

**What it does:**

- Verifies main is clean, CI is green, and the version doesn't already exist.
- Runs `make release-check` (goreleaser config).
- Generates a CHANGELOG entry from commits since last tag.
- Tags the release and pushes the tag (with explicit confirmation —
  destructive).

**Pros:**

- Release process is exactly the kind of checklist that's painful to remember
  and easy to half-finish.

**Cons:**

- Destructive (push tag); needs careful confirmation gates.
- Could just be a Makefile target. Whether this is a skill or a script is a
  judgment call.

#### `libtftest:update-design` (agent)

**Trigger:** "Update the design doc to reflect what we just built."

**What it does:**

- Reads recent commits and current state of the relevant package.
- Drafts an update to the matching DESIGN doc section (or proposes a new
  ADR/IMPL if scope warrants).
- Returns a diff for the maintainer to review.

**Pros:**

- DESIGN-0001 already drifted between original draft and final implementation;
  having a way to keep it current cheaply prevents bit-rot.

**Cons:**

- Requires good judgment about scope — should it amend DESIGN-0001 or write a
  new ADR? Risk of generating doc churn.

#### `libtftest-reviewer` (agent)

**Trigger:** Manual invocation, or via the `code-reviewer` slot in PR workflows.

**What it does:**

- Reviews changes to libtftest itself with awareness of:
  - Naming conventions (`tb` not `t`, `Seed*` for fixtures, `<Service>` for
    assert namespace vars)
  - Cleanup pairing (every Seed has a Cleanup)
  - Edition gating (Pro-only assertions call `RequirePro`)
  - Correct use of `PortEndpoint` instead of `Endpoint`
  - `BuildOptions` vs `BuildPlanOptions` separation
  - `aws.Config` passed by value, not pointer

**Pros:**

- Catches the exact mistakes we made during initial implementation. Acts as a
  safety net before human review.
- Operates in fresh context — useful for second-opinion on big PRs.

**Cons:**

- Some checks duplicate what golangci-lint and tests already cover.
- Risk of being noisy if the rules aren't tuned.

---

### Audience 2: libtftest consumers (plugin skills in `donaldgifford/claude-skills`)

These ship in a dedicated `libtftest` plugin under `donaldgifford/claude-skills`
(layout: `plugins/libtftest/skills/tftest*/` and
`plugins/libtftest/agents/tftest-reviewer/`). They activate when:

- The user mentions `libtftest` in the prompt
- The repo has `github.com/donaldgifford/libtftest` in any `go.mod`
- The user is editing a file under `test/` in a repo that also contains `*.tf`
- The user invokes them via `/tftest:<sub>`

#### `tftest`

The umbrella skill. Loaded automatically when libtftest context is detected.

**What it does:**

- Provides the model with the libtftest mental model: `TestCase`, `New()`,
  `SetVar`, `Apply`, `Plan`, `AWS()`, `Prefix()`, `RequirePro()`, three
  container lifecycle modes, sneakystack as opt-in sidecar.
- Knows about the `LIBTFTEST_*` env vars and override file naming.
- Knows the libtftest project conventions consumers should mirror: tests live
  under `test/` with a `//go:build integration` tag, libtftest itself uses
  `golangci-lint` v2 with the Uber Go Style Guide, and the `tb testing.TB`
  parameter naming convention applies to fixtures.
- Tells the model where consumer-facing docs live
  (`github.com/donaldgifford/libtftest/docs/examples/`).
- Does not generate code itself — its job is to make sure other skills (and the
  model in general) have the right vocabulary.

**Pros:**

- Without this, sub-skills have to repeat libtftest context every time.
- Lets the model answer questions ("how do I share a container across tests?")
  without invoking a generation skill.

**Cons:**

- Risk of going stale if the libtftest API changes — needs versioning (probably
  a `libtftest_version` line in frontmatter mapped to a known API surface).

#### `tftest:scaffold`

**Trigger:** "Set up libtftest tests in this repo" or `/tftest:scaffold`.

**What it does:**

- Detects the module structure (single module at root vs `modules/<name>/`
  layout vs Terragrunt-style).
- Generates `test/` directory with:
  - `go.mod` pinned to the latest libtftest minor version
  - `TestMain` using `harness.Run` (defaults to per-package container)
  - A starter `_test.go` with a single passing test against the module
  - `.gitignore` entries for `terraform.tfstate*` and `_libtftest_*`
- Optionally invokes `tftest:setup-ci` to add the GH Actions workflow.
- Prompts for: edition (Community/Pro), default services to enable, sneakystack
  sidecar yes/no.

**Pros:**

- This is the single highest-leverage skill in the consumer set — turning a
  ~200-line setup task into a one-shot.
- Handles the `hashicorp/setup-terraform@v3` requirement automatically.

**Cons:**

- Has to handle module-layout variation. We'll ship with two presets
  (single-module, multi-module under `modules/`) and prompt for anything weird.
- `go.mod` version pin needs a fetch; skill should default to "latest minor"
  with optional override.

#### `tftest:add-test`

**Trigger:** "Add a test for `<resource>`" inside a libtftest-using `test/`
directory.

**What it does:**

- Prompts for the resource type and what to assert.
- Adds a new `Test*` function to the appropriate `_test.go` (or creates one)
  using the table-driven pattern.
- Wires `tc.SetVar(...)` calls based on `variables.tf` in the module.
- Picks the matching `assert.*` helper for the resource (or notes that the
  needed assertion doesn't exist and recommends opening an upstream issue
  against libtftest).

**Pros:**

- Drives the "add one more test case" loop, which is what most consumers will do
  most of the time.

**Cons:**

- Quality depends on the model picking the right assertion — needs the `tftest`
  umbrella skill loaded for context.

#### `tftest:add-fixture`

**Trigger:** "I need to seed `<resource>` before this test runs."

**What it does:**

- Adds a `fixtures.Seed*` call to the test before `tc.Apply()`.
- If the consumer's module needs a fixture libtftest doesn't provide, generates
  a local helper function with `t.Cleanup`.
- Recommends opening an upstream PR if the fixture seems generally useful.

**Pros:**

- Keeps the fixture/cleanup pairing consistent on the consumer side too.
- Surfaces gaps in libtftest's fixture coverage organically (via the "open
  upstream PR" prompt).

**Cons:**

- Borderline overlap with `tftest:add-test` — could be folded in.

#### `tftest:add-assertion`

**Trigger:** "Assert `<thing>` in this test."

**What it does:**

- Adds an `assert.*` call after `tc.Apply()`.
- If no matching assertion exists, generates an inline assertion using
  `tc.AWS()` and the typed SDK clients, and recommends contributing an upstream
  assertion.

**Pros:**

- Same as `tftest:add-fixture`: surfaces gaps, keeps style consistent.

**Cons:**

- Same overlap risk with `tftest:add-test`.

#### `tftest:setup-ci`

**Trigger:** "Add CI for these tests" or invoked by `tftest:scaffold`.

**What it does:**

- Adds `.github/workflows/integration.yml` that calls the reusable
  `donaldgifford/libtftest/.github/workflows/libtftest-module.yml`.
- Wires `LOCALSTACK_AUTH_TOKEN` secret (if Pro) and Terraform setup.
- Adds a Codecov upload step if the repo already uses Codecov.
- Adds a status badge to the README.

**Pros:**

- The reusable workflow exists; this skill is thin glue but saves the consumer
  from reading our docs.

**Cons:**

- Repos vary widely in CI conventions (some use CircleCI, some have weird
  org-level workflows). Skill should bail out gracefully if non-GHA CI is
  detected.

#### `tftest:enable-pro`

**Trigger:** "I have a LocalStack Pro token, enable Pro features."

**What it does:**

- Updates `Options.Edition` to `localstack.EditionPro` in tests, or sets
  `LIBTFTEST_LOCALSTACK_IMAGE` in CI.
- Adds the `LOCALSTACK_AUTH_TOKEN` secret to the GH Actions workflow.
- Removes any `t.Skip` calls that were guarding Pro-only assertions.
- Adds `libtftest.RequirePro(t)` to tests that depend on Pro features.

**Pros:**

- Pro-vs-Community switchover is a clean, contained change but easy to do
  inconsistently across many tests.

**Cons:**

- Niche — only consumers with a Pro license will use it.

#### `tftest:enable-sneakystack`

**Trigger:** "Add sneakystack for SSO Admin / Organizations / Control Tower."

**What it does:**

- Adds the `sneakystack.NewSidecar` to `harness.Run` config in `TestMain`.
- Updates docs and CI workflow to pull the sneakystack image if running
  externally.
- Prompts for which gap services to enable.

**Pros:**

- sneakystack integration has multiple touchpoints (TestMain, env vars, CI);
  worth automating.

**Cons:**

- Only useful for consumers testing modules that touch SSO/Orgs/CT.

#### `tftest:debug`

**Trigger:** "My libtftest tests are failing intermittently" or "This test
passed locally but failed in CI."

**What it does:**

- Walks the artifact dump path: where logs live, how to enable trace logging
  (`LIBTFTEST_LOG_LEVEL=debug`), how to keep the LocalStack container alive for
  inspection (`LIBTFTEST_KEEP_CONTAINER=1`).
- Diagnoses common failures: `tofu` vs `terraform` PATH issue, container port
  collision, `aws.Config` cache, plan/apply state mismatch.
- Suggests narrowing reproduction (`go test -run TestX -count=1`) and inspecting
  the override files left in `_libtftest_*`.

**Pros:**

- Test flakes are the most common consumer pain. A debug skill captures the
  "where do I look first" wisdom.

**Cons:**

- Encyclopedic content tends to drift from real failure modes. Should be kept
  short and link to docs.

#### `tftest:upgrade`

**Trigger:** "Upgrade libtftest to vX.Y."

**What it does:**

- Reads the libtftest CHANGELOG between current and target version.
- Applies any mechanical migrations (e.g., renamed options, moved imports).
- Runs the test suite and reports any breakages with suggested fixes.
- Bumps `go.mod` and `go.sum`.

**Pros:**

- Pre-v1 means breaking changes are likely; making upgrades cheap is critical
  for adoption.

**Cons:**

- Quality depends entirely on the CHANGELOG being well-maintained. Couples this
  skill to libtftest release discipline.

#### `tftest-reviewer` (agent)

**Trigger:** Manual, or as a PR-review subagent.

**What it does:**

- Reviews consumer test code for:
  - Proper `tc.Prefix()` usage in resource names (parallel safety)
  - Cleanup ordering (fixtures registered before Apply)
  - Edition gating (Pro-only assertions guarded with RequirePro)
  - Coverage of key module outputs
  - No hardcoded resource names that would collide under `t.Parallel()`
  - Reasonable test data (avoid hardcoded credentials, real account IDs)

**Pros:**

- Independent context — useful for honest second opinion on test PRs.
- Catches parallel-safety bugs that don't surface until the test suite grows.

**Cons:**

- May overlap with whatever code-review tooling the consumer already has. Should
  be opt-in.

## API / Interface Changes

No code API changes. The skills themselves are the API surface. Each skill ships
with frontmatter:

```yaml
---
name: tftest:scaffold
description: Bootstrap a libtftest test directory in a Terraform module repo.
tools:
  - Read
  - Write
  - Edit
  - Bash
libtftest_version: ">=0.1.0"
---
```

The `libtftest_version` field is new (not part of standard skill schema yet)
and constrains which libtftest API versions the skill targets. The umbrella
`tftest` skill, on activation, runs `go list -m -f '{{.Version}}' github.com/donaldgifford/libtftest`
in the consumer repo and warns if the installed version doesn't satisfy the
constraint of the currently-installed plugin. This is best-effort: the warning
goes to the model, which should surface it to the user before generating code
that may not match their installed API.

## Data Model

None. Skills are markdown files.

## Testing Strategy

- **Local skills**: smoke-test by invoking each skill against a clean branch and
  verifying the generated code compiles and passes lint. Add a
  `make test-skills` target that runs each scaffold-style skill against a
  scratch dir.
- **Plugin skills**: maintain a fixtures repo (`libtftest-skill-fixtures`) with
  three sample Terraform module layouts. CI for the `libtftest` plugin runs
  each consumer skill against each fixture and verifies the result builds and
  the generated test passes.
- **Agents**: hard to test deterministically; rely on golden-file diff of agent
  output on a curated set of PRs.

## Migration / Rollout Plan

Phased so we don't try to ship 18 skills at once.

**Phase 1 — high-leverage local skills (1 week after v0.1.0)**

- `libtftest:add-assertion`
- `libtftest:add-fixture`
- `libtftest:add-awsx-client`

These three are the highest churn surfaces and the easiest to scaffold.

**Phase 2 — consumer scaffolding (2 weeks after Phase 1)**

- `tftest` (umbrella)
- `tftest:scaffold`
- `tftest:setup-ci`

Ship in the dedicated `libtftest` plugin. Marketing moment: a single skill gets
you from zero to passing CI.

**Phase 3 — consumer day-2 skills**

- `tftest:add-test`
- `tftest:add-fixture`
- `tftest:add-assertion`
- `tftest:debug`

The "I'm using libtftest, what now?" set.

**Phase 4 — review agents**

- `libtftest-reviewer`
- `tftest-reviewer`

Add once we have enough real PRs to tune the rules against.

**Phase 5 — operational skills**

- `libtftest:add-sneakystack-service`
- `libtftest:bump-localstack`
- `libtftest:release`
- `tftest:enable-pro`
- `tftest:enable-sneakystack`
- `tftest:upgrade`

Lower-frequency tasks; build when first painful.

**Deferred / maybe-never**

- `libtftest:add-localstack-default` (probably folded into existing skills)
- `libtftest:update-design` (judgment call; revisit after first major API
  change)

## Resolved Decisions

The following questions were raised during initial review and resolved before
implementation:

1. **Plugin home for consumer skills.** Ship as a dedicated `libtftest` plugin
   in `donaldgifford/claude-skills` rather than folding into the existing
   `infrastructure-as-code` plugin. Easier to test in isolation; we can merge
   into the IaC plugin later once the surface stabilizes.
2. **Version skew between consumer skills and libtftest releases.** Detect the
   installed libtftest version at skill activation and warn if it doesn't match
   the skill's `libtftest_version` constraint. May be slightly brittle to
   implement (parsing `go.mod` reliably across layouts), but it's the best UX —
   silent breakage is the worst alternative.
3. **`tftest:lint` skill.** Out of scope. The umbrella `tftest` skill encodes
   that libtftest uses `golangci-lint` v2 with the Uber Go Style Guide; that's
   sufficient context for the model to know what to run. No separate skill.
4. **Skills vs. agents for review work.** Default to agents (fresh context →
   bigger, faster reviews). Both `libtftest-reviewer` and `tftest-reviewer`
   are agents. We'll only fall back to a skill form if a specific review case
   doesn't make sense as an agent (none identified yet).
5. **End-user discoverability.** Link from `docs/examples/README.md` to the
   plugin install instructions, and add a short "Using Claude Code with
   libtftest" section to the root `README.md` that calls out the plugin.

## Open Questions

None remaining.

## References

- DESIGN-0001 §"Claude Code Automation" — original cut of the skill list
- IMPL-0001 — phases that delivered the libtftest API the skills target
- `donaldgifford/claude-skills` — home for the new dedicated `libtftest`
  plugin (consumer skills)
- Claude Code skills docs — <https://code.claude.com/docs/en/skills>
- libtftest docs/examples/ — reference content the skills should keep in sync
