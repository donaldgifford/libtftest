---
id: IMPL-0005
title: "CI hardening, release pipeline repair, and Makefile-to-just migration"
status: Completed
author: Donald Gifford
created: 2026-09-13
---

<!-- markdownlint-disable-file MD025 MD041 -->

# IMPL-0005: CI hardening, release pipeline repair, and Makefile-to-just migration

**Status:** Completed **Author:** Donald Gifford **Date:** 2026-09-13

<!--toc:start-->
- [Objective](#objective)
- [Findings that motivated this work](#findings-that-motivated-this-work)
- [Scope](#scope)
  - [In Scope](#in-scope)
  - [Out of Scope](#out-of-scope)
- [Implementation Phases](#implementation-phases)
  - [Phase 1: Release pipeline repair](#phase-1-release-pipeline-repair)
    - [Tasks](#tasks)
    - [Success Criteria](#success-criteria)
  - [Phase 2: Workflow hardening](#phase-2-workflow-hardening)
    - [Tasks](#tasks-1)
    - [Success Criteria](#success-criteria-1)
  - [Phase 3: Task runner migration (Makefile → just)](#phase-3-task-runner-migration-makefile--just)
    - [Tasks](#tasks-2)
    - [Success Criteria](#success-criteria-2)
  - [Phase 4: Docker pipeline hygiene](#phase-4-docker-pipeline-hygiene)
    - [Tasks](#tasks-3)
    - [Success Criteria](#success-criteria-3)
  - [Phase 5: Cleanup, verification, release](#phase-5-cleanup-verification-release)
    - [Tasks](#tasks-4)
    - [Success Criteria](#success-criteria-4)
- [Verification log (2026-09-13, local)](#verification-log-2026-09-13-local)
- [Post-merge verification log (2026-09-13)](#post-merge-verification-log-2026-09-13)
- [File Changes](#file-changes)
- [Decisions](#decisions)
- [Dependencies](#dependencies)
- [References](#references)
<!--toc:end-->

## Objective

Three things, landed together on `chore/cleanup` so they can be verified as a
unit:

1. **Repair the release pipeline.** `release.yml` has been unparsable since
   2026-07-03, so no release has shipped since v0.2.0 (2026-05-13).
2. **Fold in the useful parts of a sibling repo's `.github/` tree** (dropped
   into `example-github/` for comparison) and close a few gaps neither tree
   had.
3. **Finish the Makefile → `just` migration** so CI, docs, skills and the
   permission allowlist all reference exactly one task runner.

**Implements:** no design doc. Originated from a side-by-side review of
`example-github/` vs `.github/` on 2026-09-13; the findings below are the
spec.

**Outcome:** shipped as [PR #30](https://github.com/donaldgifford/libtftest/pull/30)
(squash commit `17cbb15`, merged 2026-09-13) and released as
[v0.2.1](https://github.com/donaldgifford/libtftest/releases/tag/v0.2.1) —
the first release since v0.2.0 (2026-05-13). See the
[post-merge verification log](#post-merge-verification-log-2026-09-13).

## Findings that motivated this work

F1–F8 came from the review; F9–F11 surfaced while verifying the fixes;
F12 surfaced on the first Release run after merge.

| # | Finding | How it was confirmed |
| - | ------- | -------------------- |
| F1 | `release.yml` line 86 read `step-security/harden-runner@harden-runner@<sha>` (doubled `@`). GitHub rejects the whole workflow file at parse time. | `gh run list --workflow=release.yml`: every run since PR #20 (2026-07-03) has `conclusion: failure` with **zero jobs**; `gh run view` reports "likely failed because of a workflow file issue". Runs appear on Renovate branches too, because an unparsable file's `on:` filter can't be evaluated. Last successful Release run: 2026-05-14. `actionlint` exits 0 on this. |
| F2 | `.goreleaser.yml` gained a `sboms:` block (commit f70aa8f, after v0.2.0) that shells out to `syft`, but no workflow installs it and `ubuntu-latest` does not ship it. | `git log -S'sboms:' -- .goreleaser.yml`; `git merge-base --is-ancestor` shows the commit is not in v0.2.0. CI's build job ran `goreleaser build --snapshot`, which never exercises `sboms:`. |
| F3 | Release notes say `tar -xzf sneakystack_*.tar.gz` but assets are `libtftest_<os>_<arch>.tar.gz` (no `project_name`, so goreleaser uses the repo name). | `gh release view v0.2.0 --json assets`. |
| F4 | `labeler.yml` `go:` globs (`pkg/`, `collector/`, `config/`, `exporter/`) match nothing in this repo; `repo:` points at `.goreleaser.yaml` (file is `.yml`); `docker:` glob `Dockerfile` misses `Dockerfile.sneakystack`; `labels.sh` creates an `ai` label that `labeler.yml` never assigns. | Read of both files vs `git ls-files`. |
| F5 | `justfile` `ci` recipe dropped `check-markers`; `bump-localstack`, `docs-matrix`, `test-examples`, `localstack-*` had no `just` equivalents; 65 `make …` references remained in CLAUDE.md, docs, skills and `.claude/settings.json`; `ci.yml` still called `make test-coverage`. | `grep -rn '\bmake '` across living docs. |
| F6 | `docker.just` `docker-buildx` runs `bake release`, whose target has `output = type=registry` and stub tags `:dev` + `:latest` — a laptop run overwrites `latest` on GHCR. | Read of `docker-bake.hcl` target `sneakystack-release` + `docker-metadata-action` stub. |
| F7 | `docker-bake.hcl` header said "renovate-operator", comments said `operator-release`, and `org.opencontainers.image.source` pointed at `github.com/donaldgifford/sneakystack` (not this repo). | Read. |
| F8 | `ci.yml` pinned `golangci-lint-action` to `v2.11.4` while `mise.toml` pins `2.13.2`, so local `lint` and CI disagreed. | Read. |
| F9 | Commit f70aa8f also replaced `.golangci.yml` with a template copy, resetting `gocritic.hugeParam.sizeThreshold` from 800 to 80 and dropping revive `context-as-argument` `allowTypesBefore`. Result: 45 findings across awsx/assert/fixtures that CI would have failed on. Separately, Go 1.27 flags `httputil.ReverseProxy.Director` as deprecated (SA1019) and goconst 2.13 flags three repeated strings. | `just lint` → 49 issues; `git diff bd15ee6 -- .golangci.yml`. |
| F10 | `docker-bake.hcl` local target used `platforms = ["linux/${BAKE_LOCAL_PLATFORM}"]`; that variable already contains the OS, so on macOS it expands to `linux/darwin/arm64/v8` and `docker buildx bake` fails. Local bake never worked on this machine. | `docker buildx bake --print` → "cannot parse platform specifier". |
| F11 | `docker.just` redeclared `set shell`; `just` rejects a setting redefined by an imported module, so **every** `just` recipe failed to parse. | `just --list` → "setting `shell` first set on line 8 is redefined on line 13". |
| F12 | The `GPG_PRIVATE_KEY` repository secret held the **public** key, so goreleaser's `signs:` step failed with `gpg: signing failed: No secret key` on the first Release run after merge. Not a code defect — the workflow, goreleaser config and key ID (`2E2CEA0BC2BD8D59`) were all correct. | Release job log for run 34775258159, attempt 1: `ghaction-import-gpg` printed `public key … imported` with no `secret key` line. Fixed by re-exporting with `gpg --armor --export-secret-keys` into the secret and `gh run rerun 34775258159 --failed`, which keeps the tag and `needs.bump-version.outputs.tag` and re-runs the skipped dependents. |

## Scope

### In Scope

- Fix F1–F11.
- Adopt from `example-github/`: syft install before goreleaser (release + CI
  build), `release --snapshot` in CI instead of `build --snapshot`, SBOM
  vulnerability scan → SARIF, `fetch-tags` + `git fetch --tags --force` in
  the changelog drift check, a rewritten `labeler.yml`.
- Gaps neither tree had: `concurrency` with cancel-in-progress on PR runs,
  `timeout-minutes` on the LocalStack jobs, an `actionlint` job, a loop guard
  so the bot's `chore(changelog)` commit doesn't re-trigger CI/Release.
- Cosmetics: `.yaml` → `.yml` for the three odd-one-out workflows, CODEOWNERS
  template note, labeler step name, reusable workflow's default Go version.
- Complete the `just` migration: recipe parity, Makefile + checkmake/makefmt
  removal, CI recipes, docs, skills, permission allowlist.
- Docker: guard the publishing recipes, fix bake metadata, add BuildKit cache
  mounts / `-trimpath` / version ldflags to `Dockerfile.sneakystack`, give
  `cmd/sneakystack` build-metadata variables and a `-version` flag.
- Delete the reference material (`example-github/`, `docker-bake-example.hcl`,
  `Dockerfile.example`) once captured.

### Out of Scope

- A blocking per-package coverage floor (`just coverage-gate`). Codecov's 60%
  project target stays advisory (`continue-on-error` on the upload).
- Pinning every remaining floating action tag to a SHA by hand —
  `renovate-config:ci` has `pinDigests: true` and will converge them.
- Rolling `step-security/harden-runner` out to every job.
- `changelog-regen.yml` from the example. It regenerates on every push to
  main, which races our tag-then-sync sequence in `release.yml`; ours is the
  better design and is kept.
- Refreshing the stale flat-layout "Adding a New AWS Service" section in
  `docs/development/README.md` (pre-IMPL-0004 content; separate docs PR).

## Implementation Phases

Each phase builds on the previous one. A phase is complete when all its tasks
are checked off and its success criteria are met.

---

### Phase 1: Release pipeline repair

Make `release.yml` parse again and make the next tag actually succeed end to
end.

#### Tasks

- [x] Fix the `harden-runner` `uses:` ref in `release.yml` (F1).
- [x] Add `anchore/sbom-action/download-syft` before GoReleaser in
      `release.yml` and in `ci.yml`'s build job (F2).
- [x] Switch `ci.yml` build job to
      `release --snapshot --skip=publish --skip=sign --clean` so archives,
      checksums and SBOMs are exercised on every PR (F2).
- [x] Set `project_name: sneakystack` in `.goreleaser.yml` so archive names
      match the release-notes header (F3).
- [x] Inject `main.version` / `main.commit` / `main.date` via goreleaser
      `ldflags`; add the matching vars and a `-version` flag to
      `cmd/sneakystack/main.go`.
- [x] Pass `VERSION` and `COMMIT` bake args from the release Docker job so
      published images carry the tag, not `0.0.0-dev`.

#### Success Criteria

- `actionlint` passes and, more importantly, the next push to `main` shows a
  Release run **with jobs** in `gh run list --workflow=release.yml`. ✓
  Run 34775258159 on the merge commit `17cbb15` ran all four jobs
  (Bump Version → Release → Changelog Sync → Docker) to `success`.
- `goreleaser release --snapshot --skip=publish --skip=sign --clean` succeeds
  locally and produces `dist/sneakystack_linux_amd64.tar.gz.spdx.json`. ✓
- `go run ./cmd/sneakystack -version` prints
  `sneakystack dev (commit none, built unknown)`. ✓

---

### Phase 2: Workflow hardening

Adopt what the example tree does better and close the gaps neither had.

#### Tasks

- [x] Rewrite `labeler.yml` for this repo's layout (`**/*.go`, `Dockerfile*`,
      `.goreleaser.yml`, `justfile`/`*.just`, `renovate.json5`, `.docz.yaml`,
      `cliff.toml`, `ai:` for `.claude/**` + `CLAUDE.md`) and update the
      descriptions in `scripts/labels.sh` (F4).
- [x] `changelog.yml`: `fetch-tags: true` + `git fetch --tags --force origin`
      before `git-cliff`.
- [x] `ci.yml` build job: scan the linux/amd64 archive SBOM with
      `anchore/scan-action` (`fail-build: false`) and upload SARIF under
      category `anchore-archive-sbom`; job gets `security-events: write`.
- [x] Loop guard: `on.push.paths-ignore: [CHANGELOG.md]` in `ci.yml` and
      `release.yml` so the changelog-sync bot commit doesn't trigger a full
      CI run or another Release run. (`pull_request` is left unfiltered so
      required checks never hang.)
- [x] `ci.yml`: `concurrency` group keyed on PR number / ref,
      `cancel-in-progress` only for `pull_request`.
- [x] `ci.yml`: `timeout-minutes` on `test-go`, `test-integration`,
      `test-integration-pro`, `docker-build`.
- [x] `ci.yml`: new `lint-actions` job running `actionlint` (installed via
      `jdx/mise-action` with `install_args: actionlint`).
- [x] Align `golangci-lint-action` `version:` with the `mise.toml` pin (F8).
- [x] `git mv` `codeql.yaml`, `dependabot-severity-label.yaml`,
      `trufflehog.yaml` → `.yml`; fix the glob in the `actionlint.yml`
      comment and document the F1 blind spot there.
- [x] CODEOWNERS: drop the "Replace @org/CHANGEME" template note.
- [x] `ci.yml` labeler step: rename from "Checkout code".
- [x] `libtftest-module.yml`: default `go-version` `1.26` → `1.27`.

#### Success Criteria

- `actionlint` clean across `.github/workflows/`. ✓
- `yamllint` excludes `.github/` by config; actionlint covers it. ✓
- Labeler reasoning: a PR touching only `assert/s3/s3.go` gets `go`; one
  touching `Dockerfile.sneakystack` gets `docker`; one touching
  `.claude/skills/...` gets `ai`. ✓ PR #30 (which touched all of those)
  received `go`, `docker`, `ai`, `ci`, `repo`, `documentation`,
  `dependencies` from the labeler plus `chore` from the branch prefix; the
  only hand-applied label was `patch`.

---

### Phase 3: Task runner migration (Makefile → just)

`just` becomes the only task runner. Every living reference to `make` is
updated; the Makefile and its lint/format tooling go away.

#### Tasks

- [x] `justfile`: add `check-markers`, `docs-matrix`, `test-examples`,
      `test-integration`, `test-integration-pro`, `localstack-up|down|status|logs`,
      `bump-localstack <calver>`; wire `check-markers` into `ci`; drop the
      duplicate `run-local` and the `helm.just` import (F5).
- [x] Remove the duplicate `set shell` from `docker.just` (F11).
- [x] Delete `Makefile`, `.checkmake.ini`, `.makefmt.yml`; remove `checkmake`
      and `makefmt` from `mise.toml`.
- [x] `ci.yml`: install `just` via `jdx/mise-action` (`install_args: just`)
      and call `just test-coverage`, `just test-integration`,
      `just test-examples`, `just test-integration-pro`.
- [x] `.claude/settings.json`: replace `Bash(make …)` allow entries with the
      `just` equivalents.
- [x] CLAUDE.md: Build & Development Commands block, marker-check sentence,
      LocalStack notes, skills list, CI Pipeline section (now documents the
      release flow and the F1/F2 gotchas).
- [x] `docs/development/README.md` (incl. stale "go 1.25 / 1.26" note),
      `docs/examples/README.md`.
- [x] Skills: `_preamble.md`, `add-awsx-client`, `add-assertion`,
      `add-fixture`, `add-sneakystack-service`, `bump-localstack` (+
      `references/release-notes-checklist.md`), `release`.
- [x] `tools/docgen/render.go` banner → `just docs-matrix`; regenerate
      `docs/feature-matrix.md`.
- [x] `renovate.json5` comment; `labeler.yml` `ci:` glob (`justfile`,
      `*.just` instead of `Makefile`).
- [x] Restore the two repo-specific `.golangci.yml` overrides wiped by the
      template refresh, with comments saying not to drop them again (F9).
- [x] Hoist `/var/run/docker.sock` and the LocalStack dummy credential
      `"test"` into named constants (goconst, F9).
- [x] Migrate `sneakystack.Proxy` from `Director` to `Rewrite`
      (`SetURL` + `SetXForwarded`; same routing, Host header and
      `X-Forwarded-For` semantics) (F9).

#### Success Criteria

- `grep -rn '\bmake ' CLAUDE.md docs/development docs/examples/README.md .claude`
  returns only prose mentioning the removed Makefile. ✓
- `just --list` shows every recipe the Makefile had, plus the new ones, with
  correct one-line descriptions. ✓
- `just ci` passes locally (lint + test + build + license-check +
  check-markers). ✓
- `claudelint run .claude/` passes (0 diagnostics, 7 files). ✓

---

### Phase 4: Docker pipeline hygiene

#### Tasks

- [x] `docker.just`: `[confirm]` on the two publishing recipes
      (`docker-release`, formerly `docker-buildx`, and `docker-push`);
      `--load` on the local builds; add `docker-print` for validating the
      bake file (F6).
- [x] `docker-bake.hcl`: correct header/comments, point
      `org.opencontainers.image.source` at `donaldgifford/libtftest`, better
      description, forward `COMMIT`/`DATE` args (F7); drop the broken
      `platforms` from the local target so buildx uses the daemon's native
      platform (F10).
- [x] `Dockerfile.sneakystack`: BuildKit cache mounts for `/go/pkg/mod` and
      `/root/.cache/go-build`, `-trimpath`, `ARG VERSION/COMMIT/DATE` →
      `-ldflags -X main.*`.
- [x] `.dockerignore`: add `build`, `testdata`, `*.out`, `*.test`,
      `coverage.*`, `justfile`, `*.just`; drop the unused `.forgejo`.

#### Success Criteria

- `just docker-print` validates the bake file. ✓
- `VERSION=test-local COMMIT=<sha> just docker-build` builds a linux/arm64
  image locally with the libtftest `source` label, and
  `docker run --rm --pull=never ghcr.io/donaldgifford/sneakystack:dev -version`
  prints `sneakystack test-local (commit 6c160bd, built unknown)`. ✓

---

### Phase 5: Cleanup, verification, release

#### Tasks

- [x] Delete `example-github/`, `docker-bake-example.hcl`,
      `Dockerfile.example`.
- [x] Add `dist/` (goreleaser output) to `.gitignore`.
- [x] Regenerate `CHANGELOG.md` with `git-cliff -o CHANGELOG.md` for the
      commits already on the branch.
- [x] `docz update impl` to refresh the impl index.
- [x] Commit the working tree (suggested grouping below), then re-run
      `git-cliff -o CHANGELOG.md` and commit it as
      `chore(changelog): regenerate` so the PR drift check passes (that
      prefix is skipped by `cliff.toml`, so the regen commit is invisible to
      the next run).
- [x] Open the PR labeled `patch`. Merging cuts v0.2.1, which also ships
      everything merged since May (PR #20, #22 and the Renovate bumps).
      → PR #30, merged 2026-09-13 18:38 UTC.
- [x] After merge: confirm `gh run list --workflow=release.yml` shows a run
      with jobs, the GitHub release has `sneakystack_*` assets + `.spdx.json`
      SBOMs, `ghcr.io/donaldgifford/sneakystack:0.2.1` exists, and no
      CI/Release run was triggered by the resulting `chore(changelog): sync`
      commit. → All confirmed; the first attempt failed at GPG signing (F12)
      and succeeded on `gh run rerun --failed`. Details in the
      [post-merge log](#post-merge-verification-log-2026-09-13).
- [x] After merge: run `scripts/labels.sh` once so the `ai` label and the
      refreshed descriptions exist on the repo. → `scripts/labels.sh --force`:
      created the 4 `severity:*` labels, updated 16 existing ones.

Suggested commit grouping (all on `chore/cleanup`):

1. `fix(ci): repair release workflow and goreleaser sbom/naming` —
   `release.yml`, `ci.yml` build job, `.goreleaser.yml`, `cmd/sneakystack`.
2. `ci: harden workflows (labeler, concurrency, actionlint, loop guard)` —
   the rest of `.github/`, `scripts/labels.sh`.
3. `chore(tools): finish Makefile → just migration` — `justfile`,
   `docker.just`, Makefile removal, `mise.toml`, `.claude/`, docs, skills,
   `tools/docgen`, `docs/feature-matrix.md`, `renovate.json5`.
4. `fix(lint): restore golangci overrides, goconst constants, Rewrite proxy`
   — `.golangci.yml`, `internal/dockerx`, `tf/`, `sneakystack/proxy.go`.
5. `build(docker): bake metadata, cache mounts, version args, guarded
   recipes` — `Dockerfile.sneakystack`, `docker-bake.hcl`, `.dockerignore`.
6. `docs(impl): IMPL-0005` — this document + index.
7. `chore(changelog): regenerate`.

The branch carried exactly those seven commits; PR #30 squash-merged them
as `17cbb15 fix(ci): repair release pipeline, harden workflows, finish just
migration (IMPL-0005) (#30)`.

#### Success Criteria

- Release run on the merge commit succeeds end to end (bump-version →
  release → changelog-sync → docker). ✓ Run 34775258159, after the F12
  re-run.
- No Release/CI run is triggered by the resulting `chore(changelog): sync`
  commit. ✓ `b609ac3 chore(changelog): sync v0.2.1` landed on `main` with
  zero workflow runs against it.

---

## Verification log (2026-09-13, local)

| Check | Result |
| ----- | ------ |
| `actionlint` (with shellcheck 0.11) over `.github/workflows/` | clean |
| `just --list` | parses; 30 recipes in 9 groups; `bump-localstack` doc line correct |
| `just ci` (lint → test → build → license-check → check-markers) | ✓ "CI pipeline complete"; 22 test packages ok; golangci-lint 2.13.2 0 issues |
| `just release-local` | release succeeded in 16s; 4 archives + 4 `.spdx.json` SBOMs via syft 1.50; `dist/sneakystack_linux_amd64.tar.gz.spdx.json` present (path the CI scan step expects) |
| `just docker-print` / `just docker-build` | bake validates; image built linux/arm64; `source` label = libtftest repo; `-version` prints injected `VERSION`/`COMMIT` |
| `just bump-localstack 2026.07.4` (same-version no-op) | runs, no stray `.bak`, no diff |
| `claudelint run .claude/` | 0 diagnostics, 7 files |
| `go run ./cmd/sneakystack -version` | `sneakystack dev (commit none, built unknown)` |
| `jq` on `.claude/settings.json` | valid; 31 allow entries |
| `git-cliff -o CHANGELOG.md` | unreleased section gains the 10 branch commits |

## Post-merge verification log (2026-09-13)

| Check | Result |
| ----- | ------ |
| PR #30 checks | All green: CI (Lint, Lint Workflows, Test Go, Integration Tests, Security Scan, Build incl. SBOM scan → `grype` code-scanning check, Docker Build, Label PR; Integration Tests (Pro) skipped without a token), Changelog Drift Check, PR Label Check, CodeQL, TruffleHog, License Check, Local Skills |
| PR #30 labels | `patch` (hand-applied) + `go`, `docker`, `ai`, `ci`, `repo`, `documentation`, `dependencies`, `chore` (automatic) |
| Release run 34775258159 (attempt 1) | Bump Version ✓ tagged `v0.2.1`; Release ✗ `gpg: signing failed: No secret key` (F12); Changelog Sync / Docker skipped |
| `GPG_PRIVATE_KEY` secret re-exported, `gh run rerun 34775258159 --failed` | Release ✓, Changelog Sync ✓, Docker ✓; run concluded `success` at 18:59 UTC |
| GitHub release `v0.2.1` (published 18:55 UTC) | `checksums.txt` + `checksums.txt.sig`; `sneakystack_{linux,darwin}_{amd64,arm64}.tar.gz`, each with a `.spdx.json` SBOM |
| `ghcr.io/donaldgifford/sneakystack` | Tags `0.2.1`, `0.2`, `latest`; manifest list with `linux/amd64` + `linux/arm64`; cosign keyless signatures with Rekor tlog entries. (Verified with `docker manifest inspect` — the local `gh` token lacks `read:packages`.) |
| Loop guard | `b609ac3 chore(changelog): sync v0.2.1` pushed to `main` by the bot; `gh run list` shows no CI or Release run for that SHA |
| `scripts/labels.sh --force` | Created the four `severity:*` labels (critical, high, medium, low); updated 16 existing labels (colors/descriptions) |
| Local `main` after merge | Diverged from `origin/main` (the ten pre-PR commits are only reachable as the squash); reset to `origin/main` before starting the close-out branch |

## File Changes

| File | Action | Description |
| ---- | ------ | ----------- |
| `.github/workflows/release.yml` | Modify | Fix harden-runner ref; syft install; `paths-ignore`; bake `VERSION`/`COMMIT` args |
| `.github/workflows/ci.yml` | Modify | `paths-ignore`, `concurrency`, timeouts, `lint-actions` job, `just` recipes, `release --snapshot` + syft + SBOM scan, golangci-lint version, labeler step name |
| `.github/workflows/changelog.yml` | Modify | `fetch-tags` + forced tag fetch |
| `.github/workflows/{codeql,dependabot-severity-label,trufflehog}.yml` | Rename | `.yaml` → `.yml` |
| `.github/workflows/libtftest-module.yml` | Modify | default Go `1.27` |
| `.github/labeler.yml` | Modify | Rewritten for this repo's layout; `ai` label |
| `.github/actionlint.yml`, `.github/CODEOWNERS` | Modify | Comment/template cleanup; F1 blind-spot note |
| `scripts/labels.sh` | Modify | Label descriptions match the new globs; usage path |
| `.goreleaser.yml` | Modify | `project_name: sneakystack`; version ldflags |
| `.golangci.yml` | Modify | Restore `hugeParam` 800 and `allowTypesBefore` (F9) |
| `cmd/sneakystack/main.go` | Modify | `version`/`commit`/`date` vars, `-version` flag, startup log fields |
| `sneakystack/proxy.go` | Modify | `Director` → `Rewrite` (F9) |
| `internal/dockerx/ping.go`, `tf/options.go`, `tf/override.go` | Modify | goconst constants (F9) |
| `Dockerfile.sneakystack` | Modify | Cache mounts, `-trimpath`, build-arg → ldflags |
| `docker-bake.hcl` | Modify | Correct metadata/comments; `COMMIT`/`DATE` args; native local platform (F10) |
| `docker.just` | Modify | Confirm guards, `--load`, `docker-print`; no `set shell` (F11) |
| `.dockerignore`, `.gitignore` | Modify | Tighter context; ignore `dist/` |
| `justfile` | Modify | Recipe parity with the Makefile + integration recipes |
| `Makefile`, `.checkmake.ini`, `.makefmt.yml` | Delete | Replaced by `just` |
| `mise.toml` | Modify | Drop `checkmake`, `makefmt` |
| `.claude/settings.json` | Modify | `make` → `just` allowlist |
| `CLAUDE.md`, `docs/development/README.md`, `docs/examples/README.md` | Modify | `make` → `just`; CI/release section |
| `.claude/skills/**` | Modify | `make` → `just` |
| `tools/docgen/render.go`, `docs/feature-matrix.md` | Modify | Banner text |
| `renovate.json5` | Modify | Comment |
| `CHANGELOG.md`, `docs/impl/README.md` | Modify | Regenerated |
| `example-github/`, `docker-bake-example.hcl`, `Dockerfile.example` | Delete | Reference material, captured |
| `docs/impl/0005-…md` | Create | This document |

## Decisions

- **`project_name: sneakystack`** rather than editing the release-notes
  header: the only binary is sneakystack, and the GitHub release still lands
  on `donaldgifford/libtftest` via `release.github`.
- **`paths-ignore` over a `head_commit.message` job guard** for the
  changelog loop: no run is created at all (quieter), and it is one line per
  workflow instead of one per job. Applied to `push` only.
- **`just` in CI via `jdx/mise-action` `install_args`** rather than
  `extractions/setup-just`: the repo already pins `mise-action` and Renovate
  tracks it; `install_args: just` installs only `just`, not the whole
  toolchain.
- **Single SBOM scan (linux/amd64)**: the Go module graph is the same on
  every target; scanning four archives would add noise, not signal. It is
  advisory (`fail-build: false`); govulncheck and Trivy remain the blocking
  scanners.
- **Makefile removed, not shimmed.** A `%: ; just $@` shim would keep two
  entry points alive and let them drift again (the Makefile's `run` target
  already pointed at a binary from another repo).
- **`docker-buildx` renamed to `docker-release`** so the recipe name says
  what it does (it builds the `release` bake group and pushes).
- **Lint regressions fixed in config, not code, where the rule conflicts
  with a documented repo convention** (`tb`-first `*Context` signatures,
  `aws.Config` by value). Code changed only for the real deprecation
  (`Director`) and the three repeated strings.
- **Local bake target has no `platforms`.** `BAKE_LOCAL_PLATFORM` includes
  the OS, so the previous `linux/${BAKE_LOCAL_PLATFORM}` could never have
  worked on macOS; letting buildx pick the daemon's platform is both correct
  and emulation-free.

## Dependencies

- `anchore/sbom-action` v0.24.2, `anchore/scan-action` v7.4.2 (new, pinned by
  SHA).
- `jdx/mise-action` `install_args` input (present in the pinned v4.2.5).
- `just` ≥ 1.23 for the `[confirm]` attribute (mise pins `latest`; local is
  1.58).
- Go ≥ 1.20 for `httputil.ProxyRequest` / `Rewrite` (go.mod is 1.27.1).

## References

- `example-github/` snapshot compared on 2026-09-13 (deleted in Phase 5;
  origin: the `hclkit` / `keycloak-cli` repos' `.github/`).
- [IMPL-0004](0004-module-hygiene-primitives-and-per-service-package-layout.md)
  Phase 7 — introduced `make check-markers` / `make docs-matrix`, now `just`.
- [IMPL-0003](0003-terratest-10-context-migration.md) — the `tb`-first
  `*Context` convention the revive override protects.
- [INV-0004](../investigation/0004-pro-and-oss-feature-matrix-tooling.md) —
  marker tooling the `check-markers` recipe drives.
- Release runs: <https://github.com/donaldgifford/libtftest/actions/workflows/release.yml>
- PR #30: <https://github.com/donaldgifford/libtftest/pull/30>
- Release run 34775258159: <https://github.com/donaldgifford/libtftest/actions/runs/34775258159>
- v0.2.1: <https://github.com/donaldgifford/libtftest/releases/tag/v0.2.1>
