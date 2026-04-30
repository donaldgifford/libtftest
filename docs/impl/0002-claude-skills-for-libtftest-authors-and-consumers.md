---
id: IMPL-0002
title: "Claude Skills for libtftest authors and consumers"
status: Draft
author: Donald Gifford
created: 2026-04-29
tags: [claude, skills, agents, plugin, automation]
---
<!-- markdownlint-disable-file MD025 MD041 -->

# IMPL 0002: Claude Skills for libtftest authors and consumers

**Status:** Draft
**Author:** Donald Gifford
**Date:** 2026-04-29

<!--toc:start-->
- [Objective](#objective)
- [Scope](#scope)
  - [In Scope](#in-scope)
  - [Out of Scope](#out-of-scope)
- [Implementation Phases](#implementation-phases)
  - [Phase 0: Foundation — repo wiring and plugin skeleton](#phase-0-foundation--repo-wiring-and-plugin-skeleton)
    - [Tasks](#tasks)
    - [Success Criteria](#success-criteria)
  - [Phase 1: High-leverage local skills (libtftest authors)](#phase-1-high-leverage-local-skills-libtftest-authors)
    - [Tasks](#tasks-1)
    - [Success Criteria](#success-criteria-1)
  - [Phase 2: Consumer scaffolding (tftest plugin)](#phase-2-consumer-scaffolding-tftest-plugin)
    - [Tasks](#tasks-2)
    - [Success Criteria](#success-criteria-2)
  - [Phase 3: Consumer day-2 skills](#phase-3-consumer-day-2-skills)
    - [Tasks](#tasks-3)
    - [Success Criteria](#success-criteria-3)
  - [Phase 4: Review agents](#phase-4-review-agents)
    - [Tasks](#tasks-4)
    - [Success Criteria](#success-criteria-4)
  - [Phase 5: Operational skills](#phase-5-operational-skills)
    - [Tasks](#tasks-5)
    - [Success Criteria](#success-criteria-5)
  - [Phase 6: Discovery and documentation](#phase-6-discovery-and-documentation)
    - [Tasks](#tasks-6)
    - [Success Criteria](#success-criteria-6)
- [File Changes](#file-changes)
  - [libtftest repo (donaldgifford/libtftest)](#libtftest-repo-donaldgiffordlibtftest)
  - [claude-skills repo (donaldgifford/claude-skills)](#claude-skills-repo-donaldgiffordclaude-skills)
- [Testing Plan](#testing-plan)
- [Dependencies](#dependencies)
- [Open Questions](#open-questions)
- [References](#references)
<!--toc:end-->

## Objective

Implement the Claude Code skills and agents specified in DESIGN-0002. Local
skills live in this repo under `.claude/`. Consumer skills ship in a new
dedicated `libtftest` plugin in `donaldgifford/claude-skills`. Review work
defaults to agents.

**Implements:**
[DESIGN-0002](../design/0002-claude-skills-for-libtftest-authors-and-consumers.md)

## Scope

### In Scope

- `.claude/skills/` and `.claude/agents/` content for libtftest maintainers in
  this repo (9 components total per DESIGN-0002 Audience 1)
- New `plugins/libtftest/` directory in `donaldgifford/claude-skills` with the
  consumer-facing skills and agents (11 components total per Audience 2)
- A fixtures repo (or `testdata/` directory) of representative Terraform module
  layouts used to smoke-test the consumer skills
- CI for both repos that validates the skills against fixtures
- Documentation updates: `docs/examples/README.md`, root `README.md`, and the
  plugin's own README
- Version-detection mechanism for consumer skills against the installed
  libtftest version

### Out of Scope

- A `tftest:lint` skill (resolved to "no" in DESIGN-0002)
- Skills for non-libtftest test frameworks (terratest-direct, kitchen-terraform)
- Tooling for module authors who don't write Go tests (tflint, tfsec, checkov)
- Auto-detection of LocalStack edition at scaffolding time

## Implementation Phases

Each phase builds on the previous one. A phase is complete when all its tasks
are checked off and its success criteria are met. The 5-phase rollout in
DESIGN-0002 is reordered here with a Phase 0 (foundation) prepended and a
Phase 6 (discovery) appended, since both are prerequisites for "real users
trying these out."

---

### Phase 0: Foundation — repo wiring and plugin skeleton

Create the directory structures, manifest files, system-prompt preamble, and
fixture-test infrastructure that all later phases consume. No actual skill
content yet.

#### Tasks

- [x] **libtftest repo: local skill scaffolding**
  - [x] Create `.claude/skills/` and `.claude/agents/` directories
  - [x] Add a shared preamble snippet at `.claude/skills/_preamble.md` that all
        local skills can reference (Uber Go Style Guide, stdlib-first,
        table-driven tests, `tb` not `t`, exported-symbol comments end with
        periods, line length 150)
  - [x] Update `.gitignore` to ensure `.claude/` is committed (it should not
        be ignored)
  - [x] Update `CLAUDE.md` with a "Repo Skills" section listing the skills
        to be added in later phases
- [x] **claude-skills repo: plugin skeleton for `libtftest`**
  - [x] Create `plugins/libtftest/` with the directory layout used by other
        plugins: `skills/`, `agents/`, `commands/`, `tests/`,
        `.claude-plugin/plugin.json`, `README.md`
  - [x] Author `plugin.json` with name `libtftest`, initial version `0.1.0`,
        keywords (`libtftest`, `terraform`, `localstack`, `terratest`,
        `testing`)
  - [x] Add the new plugin to the marketplace listing if there is one
        (verify whether the repo's root README needs an entry)
- [x] **Fixture infrastructure for consumer-skill testing**
  - [x] In the `claude-skills` repo, create
        `plugins/libtftest/tests/fixtures/` with three Terraform module
        layouts: (a) single-module at root, (b) `modules/<name>/` mono-repo
        layout, (c) Terragrunt-style `live/` layout
  - [x] Each fixture has minimal `*.tf` files and a placeholder `test/`
        directory the skill is expected to populate
  - [~] (deferred) `make test-libtftest-skills` end-to-end target. The
        existing `make test-plugin PLUGIN=libtftest` covers the
        structural test harness; an actual skill-vs-fixture e2e runner
        requires the `claude` CLI in non-interactive mode and is built
        out per-skill in later phases.
- [x] **Common frontmatter and the version-detection helper**
  - [x] Define the standard frontmatter shape for `tftest:*` skills using
        only recognized fields: `name`, `description`, `when_to_use`,
        `paths`, `allowed-tools`. (Resolved Decision #2: no
        `libtftest_version` field.) — captured in
        `plugins/libtftest/skills/_frontmatter.md`
  - [x] Document the supported libtftest version range as plain text in
        each skill body (e.g., "this skill targets libtftest >=0.1.0,
        <0.3.0; warn the user if the installed version is outside this
        range"). — convention documented in `_frontmatter.md`
  - [x] Implement the version-detection helper as a shared snippet at
        `plugins/libtftest/skills/_version-check.md` that runs `go list -m
        -f '{{.Version}}' github.com/donaldgifford/libtftest`, parses the
        result, and surfaces a warning string the umbrella skill emits to
        the model on activation
  - [x] Add a `tests/test.sh` skeleton that runs `claudelint run
        plugins/libtftest` plus the fixture-based integration tests

#### Success Criteria

- `.claude/skills/` and `.claude/agents/` exist in libtftest repo, both empty
  except for `_preamble.md`
- `plugins/libtftest/` exists in `donaldgifford/claude-skills` with a valid
  `plugin.json` and a stub README
- The plugin can be installed locally (e.g., via `claude plugin install
  ./plugins/libtftest`) without errors
- `make test-libtftest-skills` runs and reports "no skills to test yet" — i.e.,
  the harness is wired but expects no work
- Three Terraform fixture layouts exist and `terraform init` succeeds in each

---

### Phase 1: High-leverage local skills (libtftest authors)

Ship the three skills that scaffold the most common new-code paths in this
repo: assertions, fixtures, awsx clients. These are the lowest-risk, highest-
churn additions.

#### Tasks

- [ ] **`libtftest:add-awsx-client`** (build first — others depend on it)
  - [ ] `SKILL.md` with frontmatter (`name`, `description`, `when_to_use`,
        `tools: [Read, Write, Edit, Bash]`)
  - [ ] System prompt: prompts for service name and SDK module path,
        generates `awsx/clients.go` constructor, adds smoke test
  - [ ] Reference doc: `references/awsx-client-template.go.tmpl` showing the
        `config.WithBaseEndpoint` pattern
  - [ ] Fixture test: invoke the skill on a scratch branch for `kms`, verify
        the generated file builds and the smoke test passes
- [ ] **`libtftest:add-assertion`**
  - [ ] `SKILL.md` with frontmatter
  - [ ] System prompt: prompts for service + method list, generates
        `assert/<service>.go` (zero-size struct + package-level var pattern),
        plus `_test.go` table-driven stubs
  - [ ] If awsx client missing, prompt user to run `libtftest:add-awsx-client`
        first (do not silently invoke another skill)
  - [ ] Reference doc: `references/assertion-template.go.tmpl` based on
        `assert/s3.go`
  - [ ] Pro-vs-Community gating prompt: skill asks the user before inserting
        `RequirePro(t)`. Lookup table for known services lives in
        `references/pro-services.md`
  - [ ] Fixture test: scaffold `assert/kms.go` for two methods, build, lint
- [ ] **`libtftest:add-fixture`**
  - [ ] `SKILL.md` with frontmatter
  - [ ] System prompt: prompts for service + `Seed*` function name +
        signature, appends to `fixtures/fixtures.go` with paired `t.Cleanup`
  - [ ] Reference doc: `references/fixture-template.go.tmpl` from existing
        `SeedS3Object`
  - [ ] Enforces `tb testing.TB` parameter naming (thelper linter)
  - [ ] Fixture test: scaffold `SeedDynamoDBItem`, verify build + lint + the
        cleanup runs in a smoke test

#### Success Criteria

- Each skill has a working `SKILL.md` and at least one reference doc
- Each skill, when invoked against the libtftest repo on a scratch branch,
  produces code that passes `make lint` and `make test-pkg PKG=./<pkg>`
- The fixture tests in `make test-libtftest-skills` pass for all three
- A maintainer can add a new assertion or fixture in under 5 minutes end-to-end

---

### Phase 2: Consumer scaffolding (`tftest` plugin)

The "marketing moment" phase. After this, a Terraform module author can
install the plugin and go from zero to a passing CI in one skill invocation.

#### Tasks

- [ ] **`tftest` (umbrella skill)**
  - [ ] `SKILL.md` with frontmatter including `paths` glob
        (`**/*_test.go, **/*.tf, **/go.mod`)
  - [ ] System prompt content: libtftest mental model, `LIBTFTEST_*` env
        vars, override file naming, lifecycle modes, golangci-lint awareness
        (libtftest itself uses v2 with the Uber Go Style Guide)
  - [ ] Embeds `_version-check.md` snippet — skill runs `go list -m` on
        activation and warns the model when the installed libtftest
        version is outside the documented support range
  - [ ] Documents the supported version range as plain text in the skill
        body (Resolved Decision #2: no frontmatter field)
  - [ ] Links out to `github.com/donaldgifford/libtftest/docs/examples/` so
        the model can fetch up-to-date examples
- [ ] **`tftest:scaffold`**
  - [ ] `SKILL.md` with frontmatter
  - [ ] System prompt: detects layout (root, `modules/<name>/`, Terragrunt),
        prompts for edition + default services + sneakystack yes/no
  - [ ] Generates `test/` directory:
    - [ ] `go.mod` with libtftest version resolved at activation (Resolved
          Decision #4): the skill runs `go list -m -versions
          github.com/donaldgifford/libtftest` and pins to the highest version
          inside the supported range
    - [ ] `TestMain` with `harness.Run` defaulting to per-package container
    - [ ] Starter `_test.go` with a single S3-or-similar passing test
    - [ ] `.gitignore` entries for `terraform.tfstate*`, `_libtftest_*`
  - [ ] Calls `tftest:setup-ci` if user opts in
  - [ ] Reference docs: three layout templates under `references/layouts/`
- [ ] **`tftest:setup-ci`**
  - [ ] `SKILL.md` with frontmatter
  - [ ] System prompt: detects existing CI (GHA / Circle / other), bails
        gracefully on non-GHA, otherwise generates
        `.github/workflows/integration.yml` calling the reusable
        `donaldgifford/libtftest/.github/workflows/libtftest-module.yml`
  - [ ] Wires `LOCALSTACK_AUTH_TOKEN` secret if Pro
  - [ ] Adds `hashicorp/setup-terraform@v3` (terratest v0.56.0 needs this)
  - [ ] Adds Codecov upload step if `codecov.yml` exists
  - [ ] Adds README badge
  - [ ] Reference doc: `references/integration-workflow.yml.tmpl`

#### Success Criteria

- All three skills have working `SKILL.md` and reference docs
- For each of the three Terraform fixture layouts: invoking
  `tftest:scaffold` produces a `test/` directory whose `go test
  -tags=integration ./...` passes against a local LocalStack
- `tftest:setup-ci` produces a workflow file that passes `actionlint`
- The umbrella `tftest` skill, when activated, surfaces the libtftest version
  context to the model (verified by inspection)
- The plugin can be installed and used end-to-end on a brand-new module repo

---

### Phase 3: Consumer day-2 skills

The "I'm using libtftest, what now?" set. These run frequently against repos
already scaffolded by Phase 2.

#### Tasks

- [ ] **`tftest:add-test`**
  - [ ] `SKILL.md` with frontmatter; activates inside `test/` directories
  - [ ] System prompt: prompts for resource under test + assertions, parses
        the module's `variables.tf` to populate `tc.SetVar` calls, picks the
        correct `assert.*` helper or surfaces a gap
  - [ ] Fixture test: against the single-module fixture, generate a second
        test that exercises a different output, build, lint, run
- [ ] **`tftest:add-fixture`**
  - [ ] `SKILL.md` with frontmatter
  - [ ] System prompt: appends `fixtures.Seed*` call before `tc.Apply()`,
        registers cleanup in the right order (before Apply runs)
  - [ ] Recommends opening an upstream PR if libtftest doesn't yet have the
        seed helper for the requested service
- [ ] **`tftest:add-assertion`**
  - [ ] `SKILL.md` with frontmatter
  - [ ] System prompt: appends `assert.*` call after `tc.Apply()`, falls
        back to inline `tc.AWS()` + typed SDK call if no helper exists
  - [ ] Recommends contributing the helper upstream when generating inline
- [ ] **`tftest:debug`**
  - [ ] `SKILL.md` with frontmatter
  - [ ] System prompt: walks artifact-dump path, env vars
        (`LIBTFTEST_LOG_LEVEL`, `LIBTFTEST_KEEP_CONTAINER`,
        `LIBTFTEST_ARTIFACT_DIR`), known failure modes
        (`tofu` vs `terraform` PATH, port collisions, plan/apply mismatch,
        `aws.Config` cache)
  - [ ] Reference doc: `references/known-failures.md` linking to relevant
        `docs/examples/` pages

#### Success Criteria

- All four skills have working `SKILL.md` and at least one reference doc
- Fixture tests pass for `tftest:add-test`, `tftest:add-fixture`, and
  `tftest:add-assertion` against all three fixture layouts
- `tftest:debug`, when given a synthetic failing test (e.g., wrong port),
  surfaces the relevant remediation step within two prompts

---

### Phase 4: Review agents

Both review components run as agents (resolved in DESIGN-0002 Q4: agents by
default).

#### Tasks

- [ ] **`libtftest-reviewer` (agent in this repo)**
  - [ ] `.claude/agents/libtftest-reviewer.md` with frontmatter declaring
        the agent's name, description, and tools (`Read`, `Bash`, `Grep`)
  - [ ] System prompt: enforces only libtftest-specific rules (`tb` not
        `t`, `Seed*` cleanup pairing, `<Service>` namespace var,
        `PortEndpoint` vs `Endpoint`, `BuildOptions` vs `BuildPlanOptions`,
        `aws.Config` by value, RequirePro gating). Defers Go style /
        architecture review to existing `go-development:go-style` and
        `go-development:go-architect` agents (Resolved Decision #6) by
        explicitly recommending the user run those for deep-style review
  - [ ] Output contract: end with a JSON block of structured findings:
        `{ severity: "error"|"warn"|"info", file, line, rule, message }[]`.
        Tests parse this block; humans read the prose above it
  - [ ] Test by invoking on PR #6 and `git diff v0.1.0~..v0.1.0`;
        compare structured findings to a golden file
- [ ] **`tftest-reviewer` (agent in `libtftest` plugin)**
  - [ ] `plugins/libtftest/agents/tftest-reviewer.md` with frontmatter
  - [ ] System prompt: review checklist covering `tc.Prefix()` usage,
        cleanup ordering, edition gating, hardcoded names, fake credentials,
        coverage of key module outputs
  - [ ] Same JSON-findings output contract as `libtftest-reviewer`
        (Resolved Decision #6)
  - [ ] Test by running against the test code generated by Phase 2/3 fixture
        runs; golden-file diff against expected findings

#### Success Criteria

- Both agents activate when invoked manually or via PR-review subagent
  pattern
- Both agents emit a parseable JSON findings block at end of output
- On a curated set of "good" and "bad" test PRs (mix of clean and
  seeded-issue fixtures), `tftest-reviewer` flags ≥80% of seeded issues
  with false-positive rate ≤1 per 10 LoC
- `libtftest-reviewer` produces findings materially different from
  golangci-lint (catches semantic libtftest-specific issues, not style —
  style is delegated to `go-development:go-style`)

---

### Phase 5: Operational skills

Lower-frequency tasks. Build when the underlying operation has been done by a
human at least twice and felt repetitive.

#### Tasks

- [ ] **`libtftest:add-sneakystack-service`** (this repo)
  - [ ] `SKILL.md` with frontmatter
  - [ ] System prompt: prompts for service name + dispatch protocol
        (JSON-1.1 / REST-XML / query), scaffolds handler + routing +
        Store-typed wrapper + httptest test
  - [ ] Two reference docs: `references/jsonrpc-handler.go.tmpl`,
        `references/restxml-handler.go.tmpl`
  - [ ] Decision point: if the third dispatch style is needed, split into
        sub-skills rather than adding more branches
- [ ] **`libtftest:bump-localstack`** (this repo) — split into Makefile
      target + skill wrapper (Resolved Decision #5)
  - [ ] Add `make bump-localstack VERSION=<x>` target to top-level
        `Makefile`: sed/grep work for the pinned image string in
        `localstack/container.go`, `Dockerfile.sneakystack`,
        `docker-bake.hcl`, README, and `docs/examples/05-custom-image.md`
  - [ ] `SKILL.md` with frontmatter — wraps the Makefile target and
        provides the playbook: read LocalStack release notes, identify
        breaking changes, run integration suite, draft CHANGELOG entry,
        update examples
  - [ ] Reference doc: `references/release-notes-checklist.md` listing
        the LocalStack release-notes URL pattern and known regression
        areas (S3 MalformedXML, edge port behavior, Pro-only services)
- [ ] **`libtftest:release`** (this repo)
  - [ ] `SKILL.md` with frontmatter, tools include `Bash`
  - [ ] System prompt: verifies clean main + green CI + new version,
        runs `make release-check`, drafts CHANGELOG, requires explicit
        confirmation before tagging/pushing
  - [ ] Hard-coded refusal to push to anything other than the resolved
        upstream remote
- [ ] **`tftest:enable-pro`** (plugin)
  - [ ] `SKILL.md` with frontmatter
  - [ ] System prompt: flips `Edition`, sets `LIBTFTEST_LOCALSTACK_IMAGE`,
        adds `LOCALSTACK_AUTH_TOKEN` secret, removes redundant `t.Skip`,
        adds `RequirePro(t)` to Pro-only tests
- [ ] **`tftest:enable-sneakystack`** (plugin)
  - [ ] `SKILL.md` with frontmatter
  - [ ] System prompt: adds `sneakystack.NewSidecar` to `harness.Run`
        config, prompts for gap services, updates CI to pull the
        sneakystack image if running externally
- [ ] **`tftest:upgrade`** (plugin)
  - [ ] `SKILL.md` with frontmatter
  - [ ] System prompt: reads libtftest CHANGELOG between versions, applies
        mechanical migrations (renamed options, moved imports), bumps
        `go.mod`/`go.sum`, runs the test suite, reports breakage
  - [ ] Couples to libtftest CHANGELOG hygiene — add an item to the
        `libtftest:release` checklist requiring a CHANGELOG entry

#### Success Criteria

- All six skills have working `SKILL.md` and reference docs
- `make bump-localstack VERSION=4.5` succeeds in isolation; the
  `libtftest:bump-localstack` skill, run on a scratch worktree, drives the
  Makefile target plus produces a complete CHANGELOG draft
- `libtftest:release`, in dry-run mode, produces a tag and CHANGELOG draft
  matching the format of v0.1.0
- `tftest:upgrade`, given a synthetic v0.1.0 → v0.2.0 transition with a
  renamed option, applies the migration and the test suite passes
- Fixture tests pass for `tftest:enable-pro` and `tftest:enable-sneakystack`

---

### Phase 6: Discovery and documentation

Make sure end-users can find the skills and that maintainers know they exist.

#### Tasks

- [ ] **libtftest repo docs**
  - [ ] Add a "Using Claude Code with libtftest" section to root `README.md`
        with install instructions for the plugin
  - [ ] Add a section to `docs/examples/README.md` linking to the plugin and
        listing the consumer skills with one-line descriptions
  - [ ] Update `CLAUDE.md` "Repo Skills" section with the final skill list
  - [ ] Add a `docs/development/README.md` subsection on the local skills
        and how maintainers should use them
- [ ] **claude-skills repo docs**
  - [ ] Author `plugins/libtftest/README.md` with: install instructions,
        skill list with descriptions, version-compat table, link back to
        libtftest repo
  - [ ] Add the plugin to the top-level marketplace README/listing
  - [ ] Add a CHANGELOG.md entry for the initial 0.1.0 release of the plugin
- [ ] **Cross-repo automation**
  - [ ] Add a CI job in libtftest repo that runs the local skills against a
        scratch branch on PR (smoke test only)
  - [ ] Add a CI job in claude-skills repo that runs `make
        test-libtftest-skills` against the fixture layouts on every PR
        touching `plugins/libtftest/`

#### Success Criteria

- A first-time user landing on the libtftest README can install the plugin
  and run `tftest:scaffold` without consulting any other doc
- The plugin's README is discoverable from the marketplace listing
- CI in both repos catches regressions in the skills

---

## File Changes

### libtftest repo (`donaldgifford/libtftest`)

| File                                                  | Action | Description                                        |
| ----------------------------------------------------- | ------ | -------------------------------------------------- |
| `.claude/skills/_preamble.md`                         | Create | Shared system-prompt preamble for local skills     |
| `.claude/skills/libtftest-add-assertion/SKILL.md`     | Create | Phase 1 assertion scaffolding skill                |
| `.claude/skills/libtftest-add-assertion/references/`  | Create | Templates and Pro-services lookup                  |
| `.claude/skills/libtftest-add-fixture/SKILL.md`       | Create | Phase 1 fixture scaffolding skill                  |
| `.claude/skills/libtftest-add-fixture/references/`    | Create | Templates                                          |
| `.claude/skills/libtftest-add-awsx-client/SKILL.md`   | Create | Phase 1 awsx client skill                          |
| `.claude/skills/libtftest-add-sneakystack-service/`   | Create | Phase 5 sneakystack handler skill                  |
| `.claude/skills/libtftest-bump-localstack/SKILL.md`   | Create | Phase 5 LocalStack bump skill                      |
| `.claude/skills/libtftest-release/SKILL.md`           | Create | Phase 5 release skill                              |
| `.claude/agents/libtftest-reviewer.md`                | Create | Phase 4 review agent                               |
| `CLAUDE.md`                                           | Modify | Add "Repo Skills" section                          |
| `README.md`                                           | Modify | Add "Using Claude Code with libtftest" section     |
| `docs/examples/README.md`                             | Modify | Link to plugin and list consumer skills            |
| `docs/development/README.md`                          | Modify | Add local-skills section                           |
| `.github/workflows/skills.yml`                        | Create | CI smoke test for local skills                     |

### claude-skills repo (`donaldgifford/claude-skills`)

| File                                                              | Action | Description                                |
| ----------------------------------------------------------------- | ------ | ------------------------------------------ |
| `plugins/libtftest/.claude-plugin/plugin.json`                    | Create | Plugin manifest                            |
| `plugins/libtftest/README.md`                                     | Create | Plugin docs                                |
| `plugins/libtftest/CHANGELOG.md`                                  | Create | Plugin changelog                           |
| `plugins/libtftest/skills/tftest/SKILL.md`                        | Create | Umbrella skill                             |
| `plugins/libtftest/skills/_version-check.md`                      | Create | Shared `go list -m` snippet                |
| `plugins/libtftest/skills/tftest-scaffold/SKILL.md`               | Create | Phase 2 scaffolding skill                  |
| `plugins/libtftest/skills/tftest-scaffold/references/layouts/`    | Create | Three Terraform layout templates           |
| `plugins/libtftest/skills/tftest-setup-ci/SKILL.md`               | Create | Phase 2 CI skill                           |
| `plugins/libtftest/skills/tftest-add-test/SKILL.md`               | Create | Phase 3 test skill                         |
| `plugins/libtftest/skills/tftest-add-fixture/SKILL.md`            | Create | Phase 3 fixture skill                      |
| `plugins/libtftest/skills/tftest-add-assertion/SKILL.md`          | Create | Phase 3 assertion skill                    |
| `plugins/libtftest/skills/tftest-debug/SKILL.md`                  | Create | Phase 3 debug skill                        |
| `plugins/libtftest/skills/tftest-enable-pro/SKILL.md`             | Create | Phase 5 Pro skill                          |
| `plugins/libtftest/skills/tftest-enable-sneakystack/SKILL.md`     | Create | Phase 5 sneakystack skill                  |
| `plugins/libtftest/skills/tftest-upgrade/SKILL.md`                | Create | Phase 5 upgrade skill                      |
| `plugins/libtftest/agents/tftest-reviewer.md`                     | Create | Phase 4 review agent                       |
| `plugins/libtftest/tests/fixtures/{single,multi,terragrunt}/`     | Create | Three module-layout fixtures               |
| `plugins/libtftest/Makefile` or root `Makefile` target            | Modify | `make test-libtftest-skills`               |
| `plugins/libtftest/.github/workflows/test.yml` or root workflow   | Modify | CI smoke test                              |
| Top-level `README.md` / marketplace listing                       | Modify | List the new plugin                        |

## Testing Plan

- **Lint**: run `claudelint run plugins/libtftest` on the consumer plugin
  (the standard tool in `donaldgifford/claude-skills`); for local skills,
  add a `claudelint run .claude/` step to libtftest CI.
- **Integration-style (libtftest repo)**: a Make target that creates a
  throwaway `git worktree` (Resolved Decision #1), invokes each Phase 1 /
  Phase 5 skill non-interactively (with canned inputs), runs `make lint &&
  make test-pkg PKG=./<pkg>` on the result, and tears the worktree down
  with `git worktree remove`. Wired into `.github/workflows/skills.yml`.
- **Integration-style (claude-skills repo)**: each consumer skill runs
  against each of the three fixture layouts via `tests/test.sh`
  (auto-discovered by the plugin marketplace's `make test`). The generated
  code must `go build`, `go test -tags=integration ./...` (with LocalStack
  available), and `golangci-lint run`.
- **Agent regression suite**: a curated set of "good" and "bad" test PRs
  used as golden inputs for both reviewer agents. Each agent emits a JSON
  findings block; tests parse and diff against expected findings. Re-run
  on every system-prompt change.
- **Manual end-to-end**: before tagging the plugin v0.1.0, perform a fresh
  install on a clean machine, scaffold a new module test from scratch, and
  verify the result matches the latest libtftest examples.

## Dependencies

- **DESIGN-0002** approved as the source of truth for skill list and behavior
- **libtftest v0.1.0** tagged and published — consumer skills depend on
  resolvable Go modules
- **`donaldgifford/libtftest/.github/workflows/libtftest-module.yml`** —
  reusable workflow that `tftest:setup-ci` references
- **claude-skills repo** plugin authoring conventions: layout per Resolved
  Decision #3, recognized SKILL.md frontmatter fields only, `claudelint`
  for plugin lint, `tests/test.sh` for auto-discovered tests
- **`docz` CLI** for any further docs maintenance
- **`go-development` plugin agents** (`go-style`, `go-architect`) installed
  for the deferred-style-review pattern in Phase 4
- Local LocalStack and Docker for running the integration smoke tests

## Resolved Decisions

The following questions were raised during initial review and resolved before
implementation.

1. **Smoke-test isolation strategy.** Use `git worktree add` for local-skill
   smoke tests. Cleaner than `git clone` (no remote round-trip, no `.git`
   duplication), and `git worktree remove` makes cleanup trivial. CI will
   create a throwaway worktree per skill, invoke the skill against it,
   verify, and tear down.
2. **`libtftest_version` frontmatter is NOT a supported field.** Confirmed
   by reading the `donaldgifford/claude-skills` repo conventions: recognized
   `SKILL.md` frontmatter fields are `description`, `when_to_use`, `name`,
   `disable-model-invocation`, `allowed-tools`, `argument-hint`, `paths`,
   `effort`, `model`, `context`, `agent`, `hooks`, `user-invocable`, `shell`.
   No `libtftest_version`. **Resolution**: encode the version constraint as
   plain text in the umbrella skill's body, and have the skill itself run
   `go list -m -f '{{.Version}}' github.com/donaldgifford/libtftest` at
   activation, parse the output, and warn the model when the installed
   version falls outside the documented support range. Best-effort only —
   we cannot block invocation, but we can surface a warning the model is
   expected to relay to the user.
3. **Plugin authoring conventions in `donaldgifford/claude-skills`.**
   Confirmed layout:
   ```text
   plugins/<name>/
   ├── .claude-plugin/plugin.json     # required manifest
   ├── README.md
   ├── skills/<skill-name>/SKILL.md   # one dir per skill, references/ alongside
   ├── commands/                      # optional
   ├── agents/                        # optional
   ├── hooks/ + hooks.json            # optional
   └── tests/test.sh                  # auto-discovered by `make test`
   ```
   Plugin lint uses `claudelint` (Go binary, pinned in `mise.toml`). Repo
   `make test` auto-discovers `tests/test.sh` per plugin. Phase 0 tasks
   updated to match.
4. **Pinning libtftest version in `libtftest:scaffold` output.** Resolve at
   activation — the skill runs `go list -m -versions
   github.com/donaldgifford/libtftest` and pins the scaffolded `go.mod` to
   the highest version satisfying the documented support range. Avoids
   re-releasing the skill on every libtftest minor.
5. **`libtftest:bump-localstack` — skill or Makefile target?** Both. Add a
   `make bump-localstack VERSION=<x>` Makefile target for the mechanical
   sed/grep work (image pin, README references). The skill wraps the
   Makefile target with the reasoning steps: drafting the CHANGELOG entry,
   reading LocalStack release notes, running integration tests, updating
   examples. Makefile target is the executable; skill is the playbook.
6. **Agent test rubric.** Both review agents emit structured JSON findings
   (`{ severity, file, line, rule, message }[]`) at the end of their
   review. Test rubric is golden-file diff against expected findings on a
   curated set of test PRs (mix of clean PRs and seeded-issue PRs). For
   the pure-Go `libtftest-reviewer`, defer style/architecture analysis to
   the existing `go-development:go-style` and `go-development:go-architect`
   agents — `libtftest-reviewer` only enforces libtftest-specific rules
   (PortEndpoint, RequirePro gating, `tb` naming, BuildOptions split).
7. **Cross-repo coupling on libtftest API changes.** Skip the auto-check.
   Maintaining cross-repo CI hooks is annoying and brittle. Rely on
   `tftest:upgrade` to handle the reactive case (consumer pulls libtftest
   minor, runs upgrade skill, mechanical migrations apply, test suite
   catches drift).
8. **Naming conflict between `libtftest:*` (local) and `tftest:*`
   (consumer).** Keep the existing prefix split. The `libtftest:` prefix
   for local skills and `tftest:` prefix for consumer skills is sufficient
   to disambiguate, and the two contexts never load simultaneously
   (local skills only activate in this repo's cwd; plugin skills only
   activate when libtftest is in `go.mod` of the cwd repo). No rename
   needed.

## Open Questions

None remaining.

## References

- [DESIGN-0002](../design/0002-claude-skills-for-libtftest-authors-and-consumers.md)
  — source design
- [DESIGN-0001 §"Claude Code Automation"](../design/0001-libtftest-shared-terratest-localstack-harness-for-aws-modules.md#claude-code-automation)
  — original skill list
- [IMPL-0001](./0001-libtftest-v010-core-library-implementation.md) — the
  libtftest core library this depends on
- `donaldgifford/claude-skills` `infrastructure-as-code/skills/terratest/` —
  reference layout for consumer skills
- Claude Code skills docs — <https://code.claude.com/docs/en/skills>
- libtftest docs/examples/ — content the consumer skills should track
