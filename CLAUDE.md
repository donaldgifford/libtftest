# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

libtftest is a Go library (`github.com/donaldgifford/libtftest`) that wraps Terratest with opinionated, LocalStack-aware defaults for testing Terraform modules. It manages LocalStack container lifecycle, injects provider and backend overrides, provides pre-configured AWS SDK v2 clients, and offers parallel-safe resource naming. The goal: module authors write ~10 lines of Go instead of ~200 for integration tests.

The module also includes `sneakystack`, a Go HTTP proxy that fills gaps in LocalStack's AWS API coverage (IAM Identity Center, Organizations, Control Tower). sneakystack ships as both an importable package and a standalone Docker container (`cmd/sneakystack/`).

**Status**: IMPL-0001 (core library) merged to main. IMPL-0002 (Claude Code skills) shipped on chore/add-claude-skills + a companion feat/libtftest-plugin branch in `donaldgifford/claude-skills`. Pending: v0.1.0 tag, sneakystack service handlers (sso_admin, organizations).

- Design doc (core): `docs/design/0001-libtftest-shared-terratest-localstack-harness-for-aws-modules.md`
- Impl plan (core): `docs/impl/0001-libtftest-v010-core-library-implementation.md`
- Design doc (skills): `docs/design/0002-claude-skills-for-libtftest-authors-and-consumers.md`
- Impl plan (skills): `docs/impl/0002-claude-skills-for-libtftest-authors-and-consumers.md`
- Development guide: `docs/development/README.md`
- Examples: `docs/examples/`

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
