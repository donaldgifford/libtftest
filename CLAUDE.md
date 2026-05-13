# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

libtftest is a Go library (`github.com/donaldgifford/libtftest`) that wraps Terratest with opinionated, LocalStack-aware defaults for testing Terraform modules. It manages LocalStack container lifecycle, injects provider and backend overrides, provides pre-configured AWS SDK v2 clients, and offers parallel-safe resource naming. The goal: module authors write ~10 lines of Go instead of ~200 for integration tests.

The module also includes `sneakystack`, a Go HTTP proxy that fills gaps in LocalStack's AWS API coverage (IAM Identity Center, Organizations, Control Tower). sneakystack ships as both an importable package and a standalone Docker container (`cmd/sneakystack/`).

**Status**: IMPL-0001 (core library) merged. IMPL-0002 (skills) shipped. IMPL-0003 (terratest 1.0 context migration) shipped as v0.1.0 + v0.1.1. IMPL-0004 (per-service package layout + module hygiene primitives + `doc.go` convention + `tools/docgen` marker scanner) in progress on `inv/eks-localstack-coverage` — Phases 1–2 complete (`assert/<service>/` + `fixtures/<service>/` per-service packages with paired `*Context` shape); Phase 3 (cross-cutting docs + doc.go rollout) underway. Targets a single `v0.2.0` release.

- Design doc (core): `docs/design/0001-libtftest-shared-terratest-localstack-harness-for-aws-modules.md`
- Impl plan (core): `docs/impl/0001-libtftest-v010-core-library-implementation.md`
- Design doc (skills): `docs/design/0002-claude-skills-for-libtftest-authors-and-consumers.md`
- Impl plan (skills): `docs/impl/0002-claude-skills-for-libtftest-authors-and-consumers.md`
- Investigation (terratest 1.0 ctx): `docs/investigation/0001-terratest-10-context-variant-migration.md`
- Impl plan (terratest 1.0 ctx): `docs/impl/0003-terratest-10-context-migration.md`
- Investigation (EKS coverage / package layout): `docs/investigation/0002-eks-coverage-via-localstack.md`
- Investigation (doc.go convention): `docs/investigation/0003-package-documentation-convention-and-gomarkdoc-toolchain.md`
- Investigation (Pro/mockta marker matrix): `docs/investigation/0004-pro-and-oss-feature-matrix-tooling.md`
- Design doc (layout + hygiene primitives): `docs/design/0003-module-hygiene-primitives-and-per-service-package-layout.md`
- Impl plan (layout + hygiene primitives): `docs/impl/0004-module-hygiene-primitives-and-per-service-package-layout.md`
- Development guide: `docs/development/README.md`
- Examples: `docs/examples/`

### Context API surface (post-IMPL-0003 Phase 1)

`TestCase` exposes both context-aware and shim methods:

- `ApplyContext(ctx) *terraform.Options` / `ApplyContextE` / `Apply` / `ApplyE`
- `PlanContext(ctx) *PlanResult` / `PlanContextE` / `Plan` / `PlanE`
- `OutputContext(ctx, name) string` / `Output(name)`

Non-context methods are permanent shims that forward to the `*Context`
variant with `tb.Context()`. They are NOT marked `// Deprecated:`. The
destroy cleanup uses `context.WithoutCancel(tb.Context())` so it survives
test-end cancellation.

### Per-service package layout (post-IMPL-0004 Phases 1–2)

Assertions and fixtures live in per-service sub-packages, not the
old flat layout. Import them with aliases to coexist with the AWS
SDK packages of the same name:

```go
import (
    s3assert  "github.com/donaldgifford/libtftest/assert/s3"
    ddbassert "github.com/donaldgifford/libtftest/assert/dynamodb"
    iamassert "github.com/donaldgifford/libtftest/assert/iam"   // Pro
    ssmassert "github.com/donaldgifford/libtftest/assert/ssm"
    lambdaassert "github.com/donaldgifford/libtftest/assert/lambda"

    s3fix      "github.com/donaldgifford/libtftest/fixtures/s3"
    ssmfix     "github.com/donaldgifford/libtftest/fixtures/ssm"
    secretsfix "github.com/donaldgifford/libtftest/fixtures/secretsmanager"
    sqsfix     "github.com/donaldgifford/libtftest/fixtures/sqs"
)
```

The function name drops the service prefix (the package carries
it): `assert.S3.BucketExists` → `s3assert.BucketExists`,
`fixtures.SeedS3Object` → `s3fix.SeedObject`, etc. Every function
keeps its paired `*Context` variant from INV-0001.

The top-level `assert/` and `fixtures/` packages have no exported
surface — their `doc.go` files document the deprecation and the
per-service migration map. Shared `FakeTB` for cross-package test
reuse lives at `internal/testfake.FakeTB` /
`internal/testfake.NewFakeTB()`.

## Build & Development Commands

```bash
# Tool versions managed by mise (see mise.toml)
mise install              # Install all tool versions

# Build
make build                # Build core binary to build/bin/libtftest

# Test
make test                 # Run all tests with race detector
make test-pkg PKG=./pkg/x # Test a specific package
make test-coverage        # Tests with coverage report (coverage.out)
make test-report          # Tests with coverage, opens HTML report

# Lint & Format
make lint                 # golangci-lint (v2, config in .golangci.yml)
make lint-fix             # golangci-lint with auto-fix
make fmt                  # gofmt + goimports (local prefix: github.com/donaldgifford)

# Combined
make check                # lint + test (pre-commit)
make ci                   # lint + test + build + license-check

# Release
make release-check        # Validate goreleaser config
make release-local        # Local goreleaser snapshot (no publish)
make release TAG=v1.0.0   # Tag and push a release
```

## Architecture

Planned package layout (from DESIGN-0001):

```
libtftest/
├── libtftest.go           # Entry point: New(), TestCase, Apply, Plan
├── cmd/
│   ├── libtftest/         # CLI entry point
│   └── sneakystack/       # Standalone binary for Docker container
├── localstack/            # Container lifecycle (testcontainers-go)
├── tf/                    # terraform.Options builder, override + backend generation, workspace copy
├── awsx/                  # AWS SDK v2 client constructors
├── fixtures/              # Pre-apply data seeding (S3, DynamoDB, SSM, Secrets)
├── assert/                # Post-apply assertion helpers per service
├── harness/               # TestMain shared-container helpers, Sidecar interface
├── sneakystack/           # LocalStack gap-filling proxy (Store interface, service handlers)
└── internal/              # Naming, Docker ping, structured logging
```

Core external dependencies: `terratest`, `testcontainers-go`, `aws-sdk-go-v2`.

## Key Design Decisions

- **Provider override via `_libtftest_override.tf.json`** — JSON overlay so user `.tf` files stay untouched; Terraform merges key-by-key.
- **Backend override via `_libtftest_backend_override.tf.json`** — forces `backend "local"` to prevent modules from hitting real S3 backends during tests.
- **Three container lifecycle modes**: per-test (max isolation), per-package (shared via `harness.TestMain`), per-suite (external container via `LIBTFTEST_CONTAINER_URL`).
- **No magic Vars injection** — callers use `tc.SetVar()` with `tc.Prefix()` in resource names explicitly.
- **stdlib-first** — `slog` for logging, `errors.Join` for cleanup aggregation. No logrus/cobra/viper.
- **sneakystack as internal package** — opt-in sidecar proxy for LocalStack gaps. Uses a `Store` interface backed by plain Go maps (no external DB dependency). Also ships as a standalone Docker container.
- **Sidecar interface** — `harness.Sidecar` allows sneakystack (and future auxiliary services) to plug into the test harness lifecycle.

## Code Conventions

- Go module path: `github.com/donaldgifford/libtftest`
- Import ordering enforced by gci: stdlib, third-party, `github.com/donaldgifford/*`
- golangci-lint v2 config based on Uber Go Style Guide (see `.golangci.yml`)
- Linter relaxations for `_test.go` files: errcheck, funlen, gocyclo, gosec, etc.
- `golines` max line length: 150 chars
- Comments on exported symbols must end with periods (godot linter)
- `nolint` directives require both explanation and specific linter name
- **Every package ships a `doc.go`** — one file per package containing only the `package <name>` declaration and a godoc-compliant multi-paragraph package comment. No imports, types, or constants belong in `doc.go`. See [INV-0003](docs/investigation/0003-package-documentation-convention-and-gomarkdoc-toolchain.md) for the convention and gap analysis vs. the `go-development` plugin. Rendering tooling (`gomarkdoc`) and CI enforcement are deferred follow-ups.
- **Pro/mockta/external-dependency markers** — when a function calls `libtftest.RequirePro(tb)` (or any future equivalent gate), add a `// libtftest:requires <tag>[,<tag>...] <reason>` line to its doc comment. Tag list is comma-separated, no whitespace inside; reason is free text. Tracked under [INV-0004](docs/investigation/0004-pro-and-oss-feature-matrix-tooling.md).

## CI Pipeline

GitHub Actions (`.github/workflows/ci.yml`): lint, test-coverage (with Codecov), security scan (govulncheck + Trivy), build (goreleaser snapshot), Docker build (Bake), integration tests (requires Docker + Terraform).

Integration tests require `hashicorp/setup-terraform@v3` in CI -- terratest v0.56.0 defaults to `tofu` if `terraform` is not in PATH.

## Lint Gotchas

- `gosec G703` on paths derived from env vars (e.g. `HOME`, `XDG_CACHE_HOME`): use `//nolint:gosec // <reason>` on the `os.MkdirAll` or `os.Stat` line, not the `Getenv` line
- `errcheck` on `Close`: use `defer x.Close() //nolint:errcheck // <reason>`
- `gocritic hugeParam` on `aws.Config` (696 bytes): threshold raised to 800 in `.golangci.yml` -- AWS SDK passes `aws.Config` by value everywhere
- `nolintlint` fires if your nolint is on the wrong line -- gosec/errcheck target the specific call, not the surrounding code

## LocalStack Notes

- Default image pinned to `localstack/localstack:4.4` -- `:latest` now requires Pro auth token
- S3 CreateBucket returns MalformedXML on 4.4 with current AWS provider version (Plan works, Apply has compat issues)
- `AllServicesReady` signature is `func(io.Reader) bool` (not `func(*http.Response) bool`)

## Documentation

Uses `docz` for structured docs under `docs/` (RFC, ADR, Design, Impl, Plan, Investigation). Config in `.docz.yaml`.

## Repo Skills

Local Claude Code skills live under `.claude/skills/` (committed, team-shared). Per DESIGN-0002 / IMPL-0002, these accelerate common libtftest development workflows. See `.claude/skills/_preamble.md` for the shared conventions every local skill should follow.

Local skills (`.claude/skills/`):

- `libtftest:add-awsx-client` — scaffold a new typed AWS SDK v2 client constructor in `awsx/`
- `libtftest:add-assertion` — scaffold a new assertion namespace + methods in `assert/`
- `libtftest:add-fixture` — scaffold a new `Seed*` fixture function with paired `t.Cleanup`
- `libtftest:add-sneakystack-service` — scaffold a new gap-service handler in `sneakystack/services/` (JSON-RPC and REST-XML templates)
- `libtftest:bump-localstack` — wraps `make bump-localstack LS_VERSION=<x>` plus the playbook (release notes, CHANGELOG, integration tests)
- `libtftest:release` — release tag + CHANGELOG drafting workflow with explicit refusal conditions

Local agents (`.claude/agents/`):

- `libtftest-reviewer` — review changes for libtftest-specific rules (PortEndpoint, RequirePro, `tb` naming, BuildOptions split). Emits structured JSON findings. Defers Go style to the `go-development:go-style` agent.

Consumer-facing skills (`tftest:*`) ship in a separate `libtftest` plugin in `donaldgifford/claude-skills`, not in this repo. See [docs/examples/README.md](docs/examples/) for the consumer skill list.

Lint local skills with `claudelint run .claude/`.
