# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

libtftest is a Go library (`github.com/donaldgifford/libtftest`) that wraps Terratest with opinionated, LocalStack-aware defaults for testing Terraform modules. It manages LocalStack container lifecycle, injects provider and backend overrides, provides pre-configured AWS SDK v2 clients, and offers parallel-safe resource naming. The goal: module authors write ~10 lines of Go instead of ~200 for integration tests.

The module also includes `sneakystack`, a Go HTTP proxy that fills gaps in LocalStack's AWS API coverage (IAM Identity Center, Organizations, Control Tower). sneakystack ships as both an importable package and a standalone Docker container (`cmd/sneakystack/`).

**Status**: IMPL-0001 Phase 3 complete. Phases 1-3 done: Go module, internal packages, LocalStack container lifecycle, health/edition detection, Terraform workspace copy, provider/backend override injection, options builder. Working on Phase 4 (core TestCase API).

- Design doc: `docs/design/0001-libtftest-shared-terratest-localstack-harness-for-aws-modules.md`
- Impl plan: `docs/impl/0001-libtftest-v010-core-library-implementation.md`

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

GitHub Actions (`.github/workflows/ci.yml`): lint, test-coverage (with Codecov), security scan (govulncheck + Trivy), build (goreleaser snapshot), Docker build (Bake).

## Documentation

Uses `docz` for structured docs under `docs/` (RFC, ADR, Design, Impl, Plan, Investigation). Config in `.docz.yaml`.
