---
id: IMPL-0001
title: "libtftest v0.1.0 — Core Library Implementation"
status: Draft
author: Donald Gifford
created: 2026-04-16
tags: [terraform, terratest, localstack, testing, go, sneakystack]
---

<!-- markdownlint-disable-file MD025 MD041 -->

# IMPL-0001: libtftest v0.1.0 — Core Library Implementation

**Status:** Draft **Author:** Donald Gifford **Date:** 2026-04-16

## Objective

Implement the libtftest Go module from scaffold to v0.1.0 tag, covering the core
TestCase API, LocalStack container lifecycle, provider/backend override
injection, AWS SDK client constructors, fixture seeding, assertion helpers,
shared-container harness, sneakystack proxy with Store interface, and the
initial CI/CD pipeline.

**Implements:**
[DESIGN-0001](../design/0001-libtftest-shared-terratest-localstack-harness-for-aws-modules.md)

## Scope

### In Scope

- Go module initialization (`github.com/donaldgifford/libtftest`)
- `internal/` packages: `naming`, `dockerx`, `logx`
- `localstack/` package: container lifecycle, edition detection, health polling
- `tf/` package: workspace copy, provider + backend override rendering,
  terraform.Options construction
- Top-level `libtftest` package: `TestCase`, `New`, `SetVar`, `Apply`, `ApplyE`,
  `Plan`, `PlanE`, `Output`, `AWS`, `Prefix`, `RequirePro`, `RequireServices`
- `awsx/` package: `EndpointResolverV2` config, typed client constructors for
  S3, DynamoDB, IAM, SSM, Secrets Manager, SQS, SNS, Lambda, KMS, Kinesis, STS
- `fixtures/` package: `SeedS3Object`, `SeedDynamoItem`, `SeedSSMParameter`,
  `SeedSecret`, `SeedSQSMessage`
- `assert/` package: S3, DynamoDB, IAM, SSM, Lambda assertion helpers with
  zero-size struct namespace pattern
- `harness/` package: `Run`, `Current`, `Config`, `Sidecar` interface
- `sneakystack/` package: Store interface, maps-backed default, HTTP proxy, SSO
  Admin + Organizations service handlers, `NewSidecar`
- `cmd/sneakystack/` binary + Dockerfile
- `testdata/mod-s3/` fixture module for integration tests
- Unit tests for all packages, integration tests behind build tags
- CI pipeline (GitHub Actions), golangci-lint, goreleaser config updates
- Reusable GH Actions workflow (`libtftest-module.yml`)

### Out of Scope

- Claude Code skills (local `.claude/` and `donaldgifford/claude-skills` plugin
  — tracked separately per DESIGN-0001 Claude Code Automation section)
- v1.1+ extras: `chaos/`, `snapshot/`, `policy/`, `cost/`, `drift/`,
  `harness.TFExec`, Trivy/Checkov integration, init-hook cookbook
- Testing against real AWS
- sneakystack services beyond SSO Admin and Organizations

## Implementation Phases

Each phase builds on the previous one. A phase is complete when all its tasks
are checked off and its success criteria are met.

---

### Phase 1: Module Scaffold and Internal Packages

Establish the Go module, directory structure, internal utilities, and CI
foundation. Everything in this phase is dependency-free (no containers, no AWS
SDK) and runs in `go test ./...` with zero external requirements.

#### Tasks

- [x] Initialize Go module: `go mod init github.com/donaldgifford/libtftest`
- [x] Set Go version in `go.mod` (go 1.25 for max compatibility)
- [x] Create directory structure matching DESIGN-0001 package layout
- [x] Implement `internal/naming` package
  - [x] `Prefix(t testing.TB) string` — `"ltt-"` + 6 hex chars from hash(test
        name + pid + nanotime)
  - [x] Unit tests: determinism within a run, uniqueness across parallel calls
- [x] Implement `internal/dockerx` package
  - [x] `Ping(ctx context.Context) error` — ping Docker daemon
  - [x] Error classification: daemon down, socket not found, permission denied
  - [x] Remediation messages: `colima start`, `rancher-desktop`, `DOCKER_HOST`,
        `TESTCONTAINERS_HOST_OVERRIDE`
  - [x] Unit tests for error classification and socket path resolution
- [x] Implement `internal/logx` package
  - [x] `slog`-based structured logger scoped to test name
  - [x] `DumpArtifact(tb, artifactDir, name, data)` — writes to given dir
        and `$LIBTFTEST_ARTIFACT_DIR` if set
  - [x] `ResolveArtifactDir(tb, baseDir)` — resolves artifact directory
  - [x] Unit tests for artifact writing and path resolution
- [x] Update `Makefile` if needed (ensure `go build ./...` covers new packages)
- [x] Update `.golangci.yml` `goimports` local-prefixes (already configured)
- [x] Verify `make lint` passes with the new packages

#### Success Criteria

- `go build ./...` succeeds
- `go test ./internal/...` passes with zero external dependencies
- `make lint` passes
- `Prefix()` generates deterministic, unique 10-char strings
- `dockerx.Ping` returns a classified error with remediation message when Docker
  is unavailable

---

### Phase 2: LocalStack Container Lifecycle

Implement the `localstack/` package that manages the testcontainers-go container
lifecycle, health checking, and edition detection. This phase introduces the
first external dependency (testcontainers-go) and requires Docker to be running
for integration tests.

#### Tasks

- [x] Add `testcontainers-go` dependency: v0.42.0
- [x] Implement `localstack/edition.go`
  - [x] `Edition` type: `EditionAuto`, `EditionCommunity`, `EditionPro`
  - [x] `DetectEdition()` — checks `LOCALSTACK_AUTH_TOKEN` env var
  - [x] Unit tests
- [x] Implement `localstack/health.go`
  - [x] `AllServicesReady` response matcher — JSON-decode `/_localstack/health`,
        return true when no service is `initializing` or `error`
  - [x] `DetectEditionFromHealth(healthBody []byte) Edition` — parse edition
        field from health response
  - [x] `ParseHealth` + `HealthResponse` for cached health state
  - [x] Unit tests against fixture JSON payloads
- [x] Implement `localstack/container.go`
  - [x] `Container` struct: `ID`, `EdgeURL`, `Edition`, `Services`, unexported
        `ctr testcontainers.Container`
  - [x] `Config` struct: `Edition`, `Image`, `Services`, `AuthToken`, `InitHooks`
  - [x] `Config.ResolveImage()` — resolve image from config,
        `LIBTFTEST_LOCALSTACK_IMAGE` env var, or default to
        `localstack/localstack:latest`
  - [x] `Config.Env()` — build env map for container
  - [x] `Start(ctx, *Config) (*Container, error)` — full lifecycle with
        `dockerx.Ping` pre-check, testcontainers.Run, health wait
  - [x] `Stop(ctx) error` — terminate container
  - [x] `Container.Endpoint()` — returns edge URL
  - [x] Init hook bind mounts via `WithHostConfigModifier`
  - [x] Unit tests for Config.ResolveImage, Config.Env, services join
- [x] Implement `localstack/init_hooks.go`
  - [x] `InitHook` struct: `Name`, `Script`
  - [x] `WriteInitHooks` — writes hooks to temp dir, returns path
  - [x] Unit tests for hook writing and permissions
- [x] Create `testdata/mod-s3/` fixture Terraform module
  - [x] Minimal S3 bucket with versioning: `main.tf`, `variables.tf`
        (`bucket_name`), `outputs.tf` (`bucket_id`, `bucket_arn`)
  - [x] `provider "aws"` block (will be overridden by libtftest)
- [x] Write integration tests (`//go:build integration`)
  - [x] `TestContainerStart_Community` — start, health check, stop
  - [x] `TestContainerStart_ImageOverride` — verify env var override
  - [x] `TestEditionDetection_FromHealthEndpoint` — verify health endpoint parsing

#### Success Criteria

- `go test -tags=integration ./localstack/...` starts a LocalStack Community
  container, polls health to ready, and stops it cleanly
- Container uses the image from `LIBTFTEST_LOCALSTACK_IMAGE` when set
- Health check correctly parses service states from fixture JSON
- `dockerx.Ping` failure produces actionable error before testcontainers runs

---

### Phase 3: Terraform Workspace and Override Injection

Implement `tf/` package: workspace copy, provider override rendering, backend
override, and terraform.Options construction.

#### Tasks

- [x] Add `terratest` dependency: v0.56.0
- [x] Implement `tf/workspace.go`
  - [x] `Workspace` struct: `Dir`, `src`
  - [x] `NewWorkspace(tb testing.TB, moduleDir string) *Workspace`
  - [x] `copyTree` — `filepath.WalkDir` + `io.Copy`, symlink follow-once
  - [x] Unit tests: copy fidelity, nested dirs, original untouched
- [x] Implement `tf/override.go`
  - [x] `RenderProviderOverride(edgeURL string) ([]byte, error)` — generate
        `_libtftest_override.tf.json` from service catalog
  - [x] `RenderBackendOverride() []byte` — generate
        `_libtftest_backend_override.tf.json` with `backend "local"`
  - [x] `WriteOverrides(dir, edgeURL string) error` — write both files
  - [x] Service catalog as Go slice (21 services from DESIGN-0001)
  - [x] Unit tests: JSON validity, port substitution, all services present
- [x] Implement `tf/options.go`
  - [x] `BuildOptions(tb, workDir, vars) *terraform.Options`
  - [x] `PluginCacheDir()` — `$XDG_CACHE_HOME/libtftest/plugin-cache` with
        macOS `~/Library/Caches` fallback
  - [x] Env vars: AWS creds, TF_PLUGIN_CACHE_DIR, TF_IN_AUTOMATION
  - [x] Unit tests: env var population, cache dir creation
- [x] Unit tests cover workspace copy, override rendering, and options
      (integration tests deferred — workspace copy tested via unit tests)

#### Success Criteria

- `copyTree` produces a faithful copy including nested dirs, ignores symlink
  cycles
- Override JSON is valid, contains all declared services, uses dynamic port
- Backend override forces `backend "local"`
- `pluginCacheDir()` returns a valid writable path on both Linux and macOS
- `make lint` passes

---

### Phase 4: Core TestCase API

Wire together the top-level `libtftest` package: `TestCase`, `New`, `SetVar`,
`Apply`, `Plan`, `Output`, `AWS`, `Prefix`, `RequirePro`, `RequireServices`.
This is the primary consumer-facing API.

#### Tasks

- [x] Implement `libtftest.go`
  - [x] `TestCase` struct (fields per DESIGN-0001)
  - [x] `Options` struct (all fields per DESIGN-0001 including `AutoPrefixVars`)
  - [x] `New(tb testing.TB, opts *Options) *TestCase`
    - [x] Docker ping pre-check via `dockerx.Ping`
    - [x] Resolve image from `opts.Image` / `LIBTFTEST_LOCALSTACK_IMAGE` / default
    - [x] Check for shared container (harness.Current() TODO)
    - [x] Create workspace via `tf.NewWorkspace`
    - [x] Write overrides via `tf.WriteOverrides`
    - [x] Build `aws.Config` via `config.WithBaseEndpoint`
    - [x] Generate prefix via `naming.Prefix`
    - [x] Merge `opts.Vars` into internal vars map
    - [x] Handle `AutoPrefixVars` — inject `tc.Prefix()` into `name_prefix`
    - [x] Register `t.Cleanup` callbacks in correct LIFO order
  - [x] `SetVar(key string, val any)`
  - [x] `Apply() *terraform.Options` — `terraform init` + `terraform apply`
  - [x] `ApplyE() (*terraform.Options, error)`
  - [x] `Plan() *PlanResult` — `terraform init` + `terraform plan -out`
  - [x] `PlanE() (*PlanResult, error)`
  - [x] `PlanResult` and `PlanChanges` types — parse via `hashicorp/terraform-json`
  - [x] `Output(name string) string`
  - [x] `AWS() aws.Config`
  - [x] `Prefix() string`
- [x] Implement edition gating (in `libtftest` package)
  - [x] `RequirePro(tb testing.TB)` — checks `LOCALSTACK_AUTH_TOKEN`, `t.Skip`
  - [x] `RequireServices(tb testing.TB, services ...string)` — stub (no-op)
- [x] Implement cleanup + artifact dumping
  - [x] On failure: dump override files and plan file via `logx`
  - [x] `PersistOnFailure` support in cleanup callbacks
  - [x] Cleanup runs in LIFO order: artifacts first, destroy second, container last
- [x] Write unit tests
  - [x] `SetVar` merging
  - [x] `Prefix` getter
  - [x] `PlanChanges` parsing from fixture JSON (table-driven)
- [x] Write integration tests (`//go:build integration`)
  - [x] `TestNew_FullLifecycle` — New -> SetVar -> Plan -> AWS -> Prefix
  - [x] `TestNew_Plan` — plan-only, verify `PlanResult` fields
  - [x] `TestRequirePro_SkipsOnCommunity` — verify skip message

#### Success Criteria

- The 10-line happy path from DESIGN-0001 works end-to-end: `New` -> `SetVar` ->
  `Apply` -> `Output` -> assertion
- `Plan` returns valid `PlanResult` with parsed `PlanChanges`
- Cleanup runs in correct order: logs first, destroy second, container last
- `PersistOnFailure` keeps container alive when test fails
- `RequirePro` auto-skips on Community with clear message
- `make test` passes (unit tests only, no Docker needed)

---

### Phase 5: AWS Clients, Fixtures, and Assertions

Implement the consumer-facing helper packages: `awsx/`, `fixtures/`, `assert/`.
These are the packages module authors interact with most after `TestCase`.

#### Tasks

- [x] Add AWS SDK v2 dependencies (S3, DynamoDB, IAM, SSM, SecretsManager,
      SQS, SNS, Lambda, KMS, Kinesis, STS)
- [x] Implement `awsx/config.go`
  - [x] `New(ctx, edgeURL) (aws.Config, error)` — `config.WithBaseEndpoint`
  - [x] Unit tests: config creation, credentials, region
- [x] Implement `awsx/clients.go`
  - [x] Typed constructors: `NewS3` (path style), `NewDynamoDB`, `NewIAM`,
        `NewSSM`, `NewSecrets`, `NewSQS`, `NewSNS`, `NewLambda`, `NewKMS`,
        `NewKinesis`, `NewSTS`
  - [x] Unit tests for constructors
- [x] Implement `fixtures/` package
  - [x] `SeedS3Object` + cleanup
  - [x] `SeedSSMParameter` + cleanup (String/SecureString)
  - [x] `SeedSecret` + cleanup (force delete)
  - [x] `SeedSQSMessage` (no cleanup — messages are consumed)
- [x] Implement `assert/` package
  - [x] `s3Asserts` struct + `var S3 s3Asserts`
    - [x] `BucketExists`, `BucketHasEncryption`, `BucketHasVersioning`,
          `BucketBlocksPublicAccess`, `BucketHasTag`
  - [x] `dynamoAsserts` struct + `var DynamoDB dynamoAsserts`
    - [x] `TableExists`
  - [x] `iamAsserts` struct + `var IAM iamAsserts`
    - [x] `RoleExists`, `RoleHasInlinePolicy` — Pro-only via `RequirePro`
  - [x] `ssmAsserts` struct + `var SSM ssmAsserts`
    - [x] `ParameterExists`, `ParameterHasValue`
  - [x] `lambdaAsserts` struct + `var Lambda lambdaAsserts`
    - [x] `FunctionExists`
- [x] Raised `hugeParam` threshold to 800 to accommodate `aws.Config` (696 bytes)
      which AWS SDK passes by value

#### Success Criteria

- `awsx.New(edgeURL)` returns an `aws.Config` whose clients successfully talk to
  a running LocalStack container
- All `Seed*` functions create resources, and `t.Cleanup` removes them
- `assert.S3.BucketExists` and friends pass against `testdata/mod-s3/` output
- IAM assertions auto-skip on Community with `RequirePro` message
- `make lint` passes for all new packages

---

### Phase 6: Shared-Container Harness

Implement `harness/` package: `TestMain` helper, shared container management,
`Sidecar` interface. This enables the per-package container reuse mode that most
teams will adopt.

#### Tasks

- [x] Implement `harness/sidecar.go`
  - [x] `Sidecar` interface: `Start(ctx, localstackURL) (edgeURL, error)`,
        `Stop(ctx) error`, `Healthy(ctx) bool`
- [x] Implement `harness/testmain.go`
  - [x] `Config` struct: `Edition`, `Image`, `Services`, `Sidecars []Sidecar`
  - [x] Package-level `shared *localstack.Container` with mutex
  - [x] `Run(m *testing.M, cfg Config)` — start container, set `shared`, run
        `m.Run()`, stop container
  - [x] `Current() *localstack.Container` — return `shared` (nil if not set)
  - [x] Sidecar orchestration: start after container, collect edge URL
  - [x] Cleanup on `m.Run()` completion: stop sidecars (reverse), stop container
  - [x] `PrefixWarning` for duplicate prefix detection
  - [x] `FormatContainerInfo` for debug output
- [x] Update `libtftest.New` to call `harness.Current()` for auto-detection
- [x] Unit tests for Current(), EdgeURL(), PrefixWarning, FormatContainerInfo

#### Success Criteria

- `TestMain` with `harness.Run` starts exactly one container shared across all
  tests in the package
- `harness.Current()` returns the shared container; `libtftest.New` uses it
  automatically
- Sidecar lifecycle is orchestrated correctly: start after LocalStack, stop
  before LocalStack
- Prefix collision warning fires when a test forgets namespacing

---

### Phase 7: sneakystack Package

Implement the `sneakystack/` package: Store interface, maps-backed default, HTTP
reverse proxy, SSO Admin + Organizations service handlers, and the
`harness.Sidecar` implementation. Also build `cmd/sneakystack/` binary and
Dockerfile.

#### Tasks

- [x] Implement `sneakystack/store.go`
  - [x] `Store` interface: `Put`, `Get`, `List`, `Delete`
  - [x] `Filter` struct: `Parent`, `Tags`
  - [x] `NewMapStore() *MapStore` — `sync.RWMutex`-protected maps
  - [x] Unit tests: CRUD, not-found, empty list, concurrent access
- [x] Implement `sneakystack/proxy.go`
  - [x] `Proxy` struct: holds Store, downstream URL, service router
  - [x] `NewProxy(store, downstreamURL) (*Proxy, error)`
  - [x] HTTP handler: route by `X-Amz-Target` header, dispatch to handler
        or forward via `httputil.ReverseProxy`
  - [x] `RegisterHandler` for service prefix matching
  - [x] Unit tests: routing, forwarding, unmatched target
- [ ] Implement `sneakystack/services/sso_admin.go` (deferred to post-v0.1.0)
- [ ] Implement `sneakystack/services/organizations.go` (deferred to post-v0.1.0)
- [x] Implement `sneakystack/sidecar.go`
  - [x] `NewSidecar(cfg Config) *Sidecar`
  - [x] `Start` — create proxy, listen on ephemeral port, serve in goroutine
  - [x] `Stop` — `http.Server.Shutdown`
  - [x] `Healthy` — TCP dial check
- [x] Create `cmd/sneakystack/main.go`
  - [x] Parse flags: `--downstream`, `--port`
  - [x] Start proxy, graceful shutdown on signal
- [x] Create `Dockerfile.sneakystack`
  - [x] Multi-stage build: Go builder -> distroless
  - [x] Expose port 4567, set entrypoint
- [x] Create `docker-bake.hcl` with sneakystack targets
  - [x] `sneakystack-ci` target (linux/amd64, GHA cache)
  - [x] `sneakystack` release target (linux/amd64, linux/arm64)
  - [x] Push to `ghcr.io/donaldgifford/sneakystack`

#### Success Criteria

- `MapStore` passes all CRUD + concurrency tests
- Proxy correctly routes SSO Admin and Organizations requests to handlers
- All other requests pass through to LocalStack unmodified
- `sneakystack.NewSidecar` works with `harness.Run` end-to-end
- `cmd/sneakystack` binary builds and Docker image produces a working container
- `make lint` passes

---

### Phase 8: CI Pipeline and Release Readiness

Harden the codebase, finalize CI, prepare goreleaser config, create the reusable
GH Actions workflow, and tag v0.1.0.

#### Tasks

- [x] Update `.github/workflows/ci.yml`
  - [x] Add integration test job (`go test -tags=integration`)
  - [x] Add sneakystack Docker build job (existing docker-build job uses bake)
  - [x] Add Pro integration test job (main branch only, with
        `LOCALSTACK_AUTH_TOKEN` secret)
- [x] Create `.github/workflows/libtftest-module.yml`
  - [x] Reusable workflow (`workflow_call`) for consumer module repos
  - [x] Inputs: go-version, terraform-version, module-path
  - [x] Steps: checkout, setup Go, setup Terraform, `go test -tags=integration`
- [x] Update `.goreleaser.yml`
  - [x] Sneakystack binary only (linux/darwin, amd64/arm64)
  - [x] Updated release metadata for libtftest
- [x] Error messages reviewed — dockerx has remediation hints, classified errors
- [x] Verify `make lint` + `make test` + `make build` pass
- [x] `goreleaser check` passes
- [x] Write `README.md` with quick-start example
- [ ] Tag `v0.1.0` (to be done after merge to main)

#### Success Criteria

- `make ci` passes: lint + test + build + license-check
- Integration tests pass in CI with Docker available
- Reusable workflow `libtftest-module.yml` is callable from a consumer repo
- goreleaser snapshot produces sneakystack binaries for linux/darwin amd64/arm64
- `docker buildx bake ci` builds sneakystack image; release target pushes to
  GHCR
- `README.md` contains a working quick-start example
- v0.1.0 tag is pushed and release is published

---

## File Changes

| File                                     | Action        | Description                                              |
| ---------------------------------------- | ------------- | -------------------------------------------------------- |
| `go.mod`, `go.sum`                       | Create        | Module init with dependencies                            |
| `libtftest.go`                           | Create        | TestCase, Options, New, Apply, Plan, Output, AWS, Prefix |
| `edition.go`                             | Create        | RequirePro, RequireServices                              |
| `internal/naming/prefix.go`              | Create        | Prefix generation                                        |
| `internal/dockerx/ping.go`               | Create        | Docker daemon detection                                  |
| `internal/logx/dump.go`                  | Create        | Artifact dumping, structured logging                     |
| `localstack/container.go`                | Create        | Container Start/Stop lifecycle                           |
| `localstack/edition.go`                  | Create        | Edition type and detection                               |
| `localstack/health.go`                   | Create        | Health polling and parsing                               |
| `localstack/init_hooks.go`               | Create        | InitHook struct, mount helper                            |
| `tf/workspace.go`                        | Create        | Workspace copy with copyTree                             |
| `tf/override.go`                         | Create        | Provider + backend override rendering                    |
| `tf/options.go`                          | Create        | terraform.Options builder, pluginCacheDir                |
| `awsx/config.go`                         | Create        | BaseEndpoint config via `config.WithBaseEndpoint`        |
| `awsx/clients.go`                        | Create        | Typed client constructors                                |
| `fixtures/*.go`                          | Create        | Seed functions with cleanup                              |
| `assert/*.go`                            | Create        | Assertion helpers per service                            |
| `harness/testmain.go`                    | Create        | Run, Current, Config                                     |
| `harness/sidecar.go`                     | Create        | Sidecar interface                                        |
| `harness/parallel.go`                    | Create        | Prefix re-export, collision warning                      |
| `sneakystack/store.go`                   | Create        | Store interface + MapStore                               |
| `sneakystack/proxy.go`                   | Create        | HTTP proxy + service routing                             |
| `sneakystack/sidecar.go`                 | Create        | harness.Sidecar implementation                           |
| `sneakystack/services/*.go`              | Create        | SSO Admin, Organizations handlers                        |
| `cmd/sneakystack/main.go`                | Create        | Standalone binary entry point                            |
| `Dockerfile.sneakystack`                 | Create        | Multi-stage Docker build                                 |
| `docker-bake.hcl`                        | Create/Modify | Sneakystack CI + release targets, push to GHCR           |
| `testdata/mod-s3/`                       | Create        | Fixture Terraform module                                 |
| `.github/workflows/ci.yml`               | Modify        | Add integration + sneakystack jobs                       |
| `.github/workflows/libtftest-module.yml` | Create        | Reusable workflow                                        |
| `.goreleaser.yml`                        | Modify        | Sneakystack binary build only                            |
| `README.md`                              | Modify        | Quick-start example                                      |

## Testing Plan

- [ ] Unit tests for all exported functions in every package
- [ ] Table-driven tests for multi-input functions (override rendering, prefix
      generation, health parsing, Store CRUD)
- [ ] Integration tests behind `//go:build integration` tag for container
      lifecycle, full TestCase flow, fixtures, assertions, sneakystack proxy
- [ ] Pro integration tests behind `//go:build integration && localstack_pro`
      for IAM assertion auto-skip verification
- [ ] `testdata/mod-s3/` fixture module tested in CI on every PR
- [ ] Coverage targets: >80% core, >70% helpers

## Dependencies

| Dependency                                    | Purpose                 | Notes                              |
| --------------------------------------------- | ----------------------- | ---------------------------------- |
| `github.com/gruntwork-io/terratest`           | Terraform test runner   | Wrapping, not forking              |
| `github.com/testcontainers/testcontainers-go` | Container lifecycle     | Requires Docker daemon             |
| `github.com/aws/aws-sdk-go-v2/*`              | AWS client constructors | v2 only, `config.WithBaseEndpoint` |
| `github.com/hashicorp/terraform-json`         | Plan JSON parsing       | `PlanResult.Changes` types         |
| Docker                                        | Container runtime       | Required for integration tests     |
| Terraform CLI                                 | Plan/Apply execution    | Installed via mise or CI           |
| LocalStack                                    | AWS service emulation   | OSS default, Pro optional          |

## Decisions

1. **LocalStack service catalog source — needs investigation.** The upstream
   source for the endpoint list is unclear (could be a Python module, JSON file,
   or internal API). **Action:** Create an INV doc to investigate the upstream
   source before Phase 3. For v0.1.0, hardcode the initial list from DESIGN-0001
   and add `go generate` automation later once the source is understood.

2. **Ryuk fallback — yes.** Honor `TESTCONTAINERS_RYUK_DISABLED=true` and rely
   solely on `t.Cleanup` when Ryuk is unavailable (rootless Docker,
   Kubernetes-based CI runners). Document this in the README.

3. **sneakystack wire protocol — match what Terraform needs.** Implement only
   the request/response fields the AWS Terraform provider actually reads for
   each resource. No need for full AWS API fidelity — match the minimum needed
   for `terraform plan` + `terraform apply` to succeed for the specific
   resources under test. Error formats should match what the provider expects
   (JSON for SSO Admin and Organizations).

4. **Plan parsing — use `hashicorp/terraform-json`.** Import it directly as a
   dependency (`go get`), no vendoring needed. It provides stable types for the
   plan JSON format and is maintained by HashiCorp alongside Terraform.

5. **Integration test parallelism — trust testcontainers.** Run fully parallel
   and trust ephemeral port mapping. Adjust to `-parallel 1` only if we hit
   actual contention issues in CI.

6. **Single tag, goreleaser for binary, docker-bake for container.** One
   `v0.x.y` tag covers both the Go module and sneakystack artifacts. Goreleaser
   builds the sneakystack CLI binary (linux/darwin, amd64/arm64). A
   `docker-bake.hcl` handles the sneakystack container image and pushes to GHCR.
   The Go library itself needs no binary — consumers `go get` it.

## Open Questions

1. **LocalStack service catalog upstream source.** Needs an investigation doc
   (INV) to determine the exact upstream artifact to parse for `go generate`.
   Blocked until Phase 3 — hardcoded list is sufficient for v0.1.0.

## References

- [DESIGN-0001](../design/0001-libtftest-shared-terratest-localstack-harness-for-aws-modules.md)
  — libtftest design doc (source of truth)
- [LocalStack Terraform Integration](https://docs.localstack.cloud/user-guide/integrations/terraform/)
- [Terratest Documentation](https://terratest.gruntwork.io/)
- [testcontainers-go Documentation](https://golang.testcontainers.org/)
- [AWS SDK Go v2 — Endpoint Resolution](https://aws.github.io/aws-sdk-go-v2/docs/configuring-sdk/endpoints/)
