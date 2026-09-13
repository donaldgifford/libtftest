# Development Guide

This guide covers how to develop, test, and contribute to libtftest and
sneakystack.

## Prerequisites

Install tool versions via [mise](https://mise.jdx.dev/):

```bash
mise install
```

This installs Go, golangci-lint, goimports, `just`, and other tools defined
in `mise.toml`. You also need Docker running for integration tests.

All project automation goes through [`just`](https://just.systems): the
`justfile` (plus `docker.just` for image builds) is the single task runner.
Run `just` with no arguments to list every recipe.

## Build and Test

```bash
# Build all binaries
just build

# Run unit tests (no Docker required)
just test

# Run a single package's tests
just test-pkg ./localstack

# Run tests with coverage report
just test-coverage

# Run tests with coverage + open HTML report
just test-report

# Run integration tests (requires Docker + Terraform)
just test-integration

# Run the docs/examples runnable tests (requires Docker + Terraform)
just test-examples
```

## Code Quality

```bash
# Lint (golangci-lint v2, config in .golangci.yml)
just lint

# Lint with auto-fix
just lint-fix

# Lint GitHub Actions workflows (actionlint)
just lint-actions

# Format (gofmt + goimports)
just fmt

# Fail if a RequirePro caller lacks a `libtftest:requires` marker
just check-markers

# Regenerate docs/feature-matrix.md from the markers
just docs-matrix

# Full CI pipeline: lint + test + build + license-check + check-markers
just ci

# Quick pre-commit check: lint + test
just check
```

## Project Structure

```
libtftest/
├── libtftest.go              # Core API: TestCase, New, Apply, Plan
├── cmd/
│   ├── libtftest/            # CLI entry point
│   └── sneakystack/          # Standalone proxy binary
├── localstack/               # Container lifecycle, health, edition detection
├── tf/                       # Workspace copy, overrides, terraform.Options
├── awsx/                     # AWS SDK v2 client constructors
├── fixtures/                 # Pre-apply data seeding
├── assert/                   # Post-apply assertion helpers
├── harness/                  # Shared-container TestMain, Sidecar interface
├── sneakystack/              # Gap-filling proxy, Store interface
│   └── services/             # Service handlers (SSO Admin, Organizations)
├── internal/
│   ├── naming/               # Parallel-safe prefix generation
│   ├── dockerx/              # Docker daemon detection
│   └── logx/                 # Structured logging, artifact dumping
├── testdata/
│   └── mod-s3/               # Fixture Terraform module for tests
└── docs/
    ├── development/          # This guide
    ├── examples/             # Usage examples
    ├── design/               # Design documents (docz)
    └── impl/                 # Implementation plans (docz)
```

## Code Conventions

- **Go module path:** `github.com/donaldgifford/libtftest`
- **Go version:** the `go` directive in `go.mod` is authoritative (1.27.x at
  the time of writing); mise pins the matching toolchain
- **Style:** Uber Go Style Guide, enforced by golangci-lint v2
- **Import ordering:** stdlib, third-party, `github.com/donaldgifford/*`
  (enforced by gci)
- **Line length:** 150 chars max (golines)
- **Comments:** exported symbols must end with periods (godot)
- **nolint directives:** require both a specific linter name and an explanation
- **Error wrapping:** `fmt.Errorf("context: %w", err)` -- skip "failed to"
- **Testing:** table-driven tests, `t.Parallel()` where possible, `t.Helper()`
  in all test helpers
- **Naming:** `tb` for `testing.TB` parameters (not `t`)

## Adding a New AWS Service

### 1. Add an `awsx` client constructor

In `awsx/clients.go`:

```go
func NewMyService(cfg aws.Config) *myservice.Client {
    return myservice.NewFromConfig(cfg)
}
```

### 2. Add assertion helpers

Create `assert/myservice.go`:

```go
package assert

type myServiceAsserts struct{}

var MyService myServiceAsserts

func (myServiceAsserts) ResourceExists(tb testing.TB, cfg aws.Config, name string) {
    tb.Helper()
    // ... SDK call + assertion
}
```

Register in `assert/assert.go`:

```go
var MyService myServiceAsserts
```

### 3. Add fixture seeders (if needed)

In `fixtures/fixtures.go`, add a `SeedMyResource` function with `t.Cleanup`.

### 4. Run lint and tests

```bash
just fmt && just lint && just test
```

## Adding a sneakystack Service Handler

sneakystack fills gaps in LocalStack's API coverage. To add a new service:

### 1. Create the handler

Create `sneakystack/services/myservice.go`:

```go
package services

type MyServiceHandler struct {
    store sneakystack.Store
}

func NewMyServiceHandler(store sneakystack.Store) *MyServiceHandler {
    return &MyServiceHandler{store: store}
}

func (h *MyServiceHandler) Handle(w http.ResponseWriter, r *http.Request) {
    target := r.Header.Get("X-Amz-Target")
    // Parse action from target, dispatch to methods
}
```

### 2. Register in the proxy

In `sneakystack/sidecar.go`, register the handler:

```go
proxy.RegisterHandler("MyServicePrefix", services.NewMyServiceHandler(store))
```

### 3. Write tests

Write unit tests with fixture request/response pairs. Only implement the fields
the Terraform AWS provider reads for the specific resources.

## Integration Tests

Integration tests are gated behind build tags:

| Tag | What it tests | When it runs |
| --- | --- | --- |
| `integration` | Full lifecycle with Community LocalStack | Every PR (CI) |
| `integration && localstack_pro` | Pro-only features (IAM enforcement, etc.) | Main branch only |

```bash
# Run integration tests locally
just test-integration

# Pro suite (needs LOCALSTACK_AUTH_TOKEN)
just test-integration-pro

# Run a specific integration test
go test -tags=integration -v -run TestNew_Plan ./...
```

### Running LocalStack locally with `lstk`

The integration tests manage their own LocalStack container via
testcontainers-go, so you don't normally start one by hand. For iterative
local work — or the per-suite mode where a single external container serves
the whole run — use the [`lstk`](https://github.com/localstack/lstk) CLI
(pinned in `mise.toml`) through the justfile:

```bash
just localstack-up       # lstk start
export LIBTFTEST_CONTAINER_URL=http://localhost:4566   # reuse it for the suite
just localstack-status   # lstk status
just localstack-logs     # lstk logs
just localstack-down     # lstk stop
```

LocalStack now ships a **single image** using calendar versioning
(`YYYY.MM.patch`, e.g. `2026.06.1`) — there is no separate `-pro` image — but
it **requires `LOCALSTACK_AUTH_TOKEN` even for free-tier use** (without one the
container exits with code 55). libtftest's default is therefore token-aware:
export a token before `just localstack-up` / the tests to get the single image
(`2026.06.1`, which also unlocks Pro), or run token-free against the last
community image (`localstack/localstack:4.14`) with no account required. Set
the image explicitly via `LIBTFTEST_LOCALSTACK_IMAGE` to override either way.

## Release Process

Releases use a single `v0.x.y` tag that covers both the Go module and
sneakystack artifacts. The normal path is label-driven: every PR carries
exactly one of `major` / `minor` / `patch` / `dont-release` (enforced by
`pr-labels.yml`), and on merge `release.yml` bumps, tags, runs goreleaser,
regenerates `CHANGELOG.md`, and builds + signs the container image.

```bash
# Validate goreleaser config
just release-check

# Local snapshot (no publish, no sign; needs syft for the SBOMs)
just release-local

# Manual tag + push (escape hatch; release.yml normally tags on merge)
just release v0.1.0
```

- **goreleaser** builds the sneakystack binary (linux/darwin, amd64/arm64),
  archives it as `sneakystack_<os>_<arch>.tar.gz` with an SPDX SBOM per
  archive, and GPG-signs the checksums
- **docker-bake.hcl** builds the sneakystack container image and pushes to
  `ghcr.io/donaldgifford/sneakystack` (`just docker-build` for a local,
  non-publishing build; `just docker-print` to validate the bake file)

## Environment Variables

| Variable | Purpose |
| --- | --- |
| `LIBTFTEST_LOCALSTACK_IMAGE` | Override the default LocalStack container image |
| `LIBTFTEST_PERSIST_ON_FAILURE` | Keep container alive on test failure for debugging |
| `LIBTFTEST_ARTIFACT_DIR` | Additional directory for CI artifact collection |
| `LOCALSTACK_AUTH_TOKEN` | LocalStack auth token — required by the unified single image; selects it over the token-free community image and enables Pro |
| `TESTCONTAINERS_RYUK_DISABLED` | Disable Ryuk reaper (for rootless Docker / K8s runners) |
| `DOCKER_HOST` | Custom Docker socket path |

## Documentation System

We use [docz](https://github.com/donaldgifford/docz) for structured
documentation. Config is in `.docz.yaml`.

```bash
# Create a new document
docz create <type> "Title"    # type: rfc, adr, design, impl, plan, investigation

# Update indexes
docz update <type>

# List documents
docz list
```

## Local Claude Code Skills

This repo ships local Claude Code skills under `.claude/skills/` and
agents under `.claude/agents/` that scaffold the most common new-code
paths in libtftest and review changes for libtftest-specific rules.
See the [Repo Skills section in CLAUDE.md](../../CLAUDE.md#repo-skills)
for the full list.

### Using the skills

Inside this repo, ask Claude Code things like:

- "Add an awsx client for cloudwatch" — invokes `libtftest:add-awsx-client`
- "Add a KMS assertion helper" — `libtftest:add-assertion` (which can
  chain to `libtftest:add-awsx-client` if the AWS client is missing)
- "Bump LocalStack to 2026.06.1" — `libtftest:bump-localstack` (which runs
  `just bump-localstack 2026.06.1`; routine bumps are handled by Renovate)
- "Tag a v0.2.0 release" — `libtftest:release`

The skills always run lint (`just lint`) and tests for the affected
package before declaring success.

### Adding a new local skill

1. Create `.claude/skills/<skill-name>/SKILL.md` with frontmatter (name,
   description, when_to_use, allowed-tools).
2. Reference `.claude/skills/_preamble.md` in the body for project
   conventions.
3. Add reference templates under `.claude/skills/<skill-name>/references/`.
4. Run `claudelint run .claude/` to verify.
5. Update the Repo Skills section in `CLAUDE.md`.

### Linting

```bash
# Install claudelint (already pinned in donaldgifford/claude-skills mise.toml)
mise install github:donaldgifford/claudelint

# Lint all local skills and agents
claudelint run .claude/
```
