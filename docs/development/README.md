# Development Guide

This guide covers how to develop, test, and contribute to libtftest and
sneakystack.

## Prerequisites

Install tool versions via [mise](https://mise.jdx.dev/):

```bash
mise install
```

This installs Go, golangci-lint, goimports, and other tools defined in
`mise.toml`. You also need Docker running for integration tests.

## Build and Test

```bash
# Build all binaries
make build

# Run unit tests (no Docker required)
make test

# Run a single package's tests
make test-pkg PKG=./localstack

# Run tests with coverage report
make test-coverage

# Run tests with coverage + open HTML report
make test-report

# Run integration tests (requires Docker)
go test -tags=integration -v -race ./...
```

## Code Quality

```bash
# Lint (golangci-lint v2, config in .golangci.yml)
make lint

# Lint with auto-fix
make lint-fix

# Format (gofmt + goimports)
make fmt

# Full CI pipeline: lint + test + build + license-check
make ci

# Quick pre-commit check: lint + test
make check
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
- **Go version:** `go 1.25` in `go.mod` (local dev uses 1.26.x via mise)
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
make fmt && make lint && make test
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
go test -tags=integration -v -race ./...

# Run a specific integration test
go test -tags=integration -v -run TestNew_Plan ./...
```

## Release Process

Releases use a single `v0.x.y` tag that covers both the Go module and
sneakystack artifacts.

```bash
# Validate goreleaser config
make release-check

# Local snapshot (no publish)
make release-local

# Tag and push a release
make release TAG=v0.1.0
```

- **goreleaser** builds the sneakystack binary (linux/darwin, amd64/arm64)
- **docker-bake.hcl** builds the sneakystack container image and pushes to
  `ghcr.io/donaldgifford/sneakystack`

## Environment Variables

| Variable | Purpose |
| --- | --- |
| `LIBTFTEST_LOCALSTACK_IMAGE` | Override the default LocalStack container image |
| `LIBTFTEST_PERSIST_ON_FAILURE` | Keep container alive on test failure for debugging |
| `LIBTFTEST_ARTIFACT_DIR` | Additional directory for CI artifact collection |
| `LOCALSTACK_AUTH_TOKEN` | LocalStack Pro auth token (enables Pro edition) |
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
