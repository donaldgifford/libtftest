---
id: DESIGN-0001
title: "libtftest — Shared Terratest + LocalStack Harness for AWS Modules"
status: Draft
author: Donald Gifford
created: 2026-04-16
tags: [terraform, terratest, localstack, testing, go, sneakystack]
---

<!-- markdownlint-disable-file MD025 MD041 -->

# DESIGN-0001: libtftest — Shared Terratest + LocalStack Harness for AWS Modules

## Overview

`libtftest` is a Go module (`github.com/donaldgifford/libtftest`) that wraps
Terratest with opinionated, LocalStack-aware defaults so Terraform module
authors can write integration tests in ~10 lines of Go instead of ~200. It owns
the LocalStack container lifecycle (start / health-check / destroy), injects
provider overrides into the module under test, hands back pre-configured AWS SDK
v2 clients for assertions, and provides parallel-safe resource naming. The
package assumes a Pro auth token is available (to cover IAM enforcement, EKS,
full RDS, ECS, Cognito, etc.) but cleanly degrades to Community edition for
services it supports.

The module also includes `sneakystack`, a Go HTTP proxy that fills gaps in
LocalStack's AWS API coverage (IAM Identity Center, Organizations, Control
Tower, etc.). sneakystack ships as both an importable Go package and a
standalone Docker container.

The success criterion is simple: a module author adds `libtftest` to
`test/go.mod`, writes a `TestFoo(t)` with `libtftest.New` + `Apply`, runs
`go test ./test/...`, and gets a green bar without touching Docker, provider
blocks, or cleanup code.

## Goals and Non-Goals

### Goals

- **Zero-boilerplate container lifecycle** — `libtftest.New(t, opts)` starts
  LocalStack, registers `t.Cleanup`, and is safe under `t.Parallel()`.
- **Drop-in Terratest compatibility** — return a `*terraform.Options` users can
  pass to existing Terratest helpers (`InitAndApply`, `Destroy`, `OutputAll`).
- **Provider-override injection** — module authors do _not_ add LocalStack
  blocks to their real `.tf` files; the harness writes a `_override.tf.json`
  into a scratch copy.
- **Backend isolation** — the harness forces `backend "local"` via a generated
  override to prevent modules from accidentally hitting a real S3 backend during
  tests.
- **Pro/Community parity** — a single API works for both editions; Pro-only
  tests are gated by a helper that `t.Skip()`s when no auth token is present.
- **Fast feedback loops** — container reuse across a package (TestMain mode),
  parallel test isolation via name prefixes, and a plugin-cache dir so
  `terraform init` doesn't re-download providers on every test.
- **Debuggability** — on failure, dump LocalStack logs, the rendered provider
  override, and the Terraform plan file as test artifacts.
- **LocalStack gap coverage** — the `sneakystack` package proxies LocalStack and
  handles services it doesn't support, so platform-team modules (IAM Identity
  Center, Organizations, Control Tower) are testable without skips.

### Non-Goals

- Testing against real AWS. A thin `real/` sibling package may land later, but
  this design is LocalStack-only.
- Replacing Terratest. We wrap, we do not fork.
- Covering every LocalStack service. We ship helpers for the high-traffic ones
  (S3, DynamoDB, IAM, SSM, Secrets Manager, SQS, SNS, Lambda, Kinesis, KMS,
  EventBridge, STS) and let callers drop down to raw SDK clients for the rest.
- Filling _all_ gaps in LocalStack's AWS surface area. sneakystack targets the
  control-plane APIs Donald's platform team uses most. Services beyond that
  scope are added incrementally as demand arises.
- State-backend testing. Terraform state stays on local disk under
  `t.TempDir()`; verifying remote-state modules is out of scope for v1.

## Package Layout

Go module: `github.com/donaldgifford/libtftest`

```
libtftest/                                  # module root
├── libtftest.go                            # top-level: New, TestCase, Apply, Destroy
├── cmd/
│   ├── libtftest/main.go                   # CLI entry point (if needed)
│   └── sneakystack/main.go                 # standalone binary for Docker container
├── localstack/
│   ├── container.go                        # testcontainers-go lifecycle
│   ├── edition.go                          # Community vs Pro detection
│   ├── health.go                           # /_localstack/health polling
│   └── init_hooks.go                       # /etc/localstack/init/ready.d/ helpers
├── tf/
│   ├── options.go                          # terraform.Options builder
│   ├── override.go                         # provider + backend override JSON generation
│   └── workspace.go                        # scratch dir + module copy
├── awsx/
│   ├── config.go                           # aws.Config pointed at LocalStack
│   └── clients.go                          # typed client constructors
├── fixtures/                               # pre-apply data seeding
│   ├── s3.go
│   ├── dynamodb.go
│   ├── ssm.go
│   └── secrets.go
├── assert/                                 # post-apply assertions
│   ├── s3.go
│   ├── iam.go
│   ├── dynamodb.go
│   └── lambda.go
├── harness/
│   ├── testmain.go                         # shared-container TestMain helper
│   ├── sidecar.go                          # Sidecar interface definition
│   └── parallel.go                         # naming + isolation primitives
├── sneakystack/                            # LocalStack gap-filling proxy
│   ├── proxy.go                            # HTTP reverse proxy + service routing
│   ├── store.go                            # Store interface + maps-backed default
│   ├── sidecar.go                          # harness.Sidecar implementation
│   └── services/
│       ├── sso_admin.go                    # IAM Identity Center permission sets
│       ├── organizations.go                # Organizations accounts, OUs
│       └── ...                             # additional gap services
└── internal/
    ├── naming/                             # deterministic-random prefix
    ├── dockerx/                            # docker ping + error classification
    └── logx/                               # structured logs, artifact dumping
```

Conventions: stdlib-first, interface-driven, minimal deps. External dependencies
are limited to:

- `github.com/gruntwork-io/terratest` — the thing we're wrapping
- `github.com/testcontainers/testcontainers-go` — container lifecycle
- `github.com/aws/aws-sdk-go-v2/*` — AWS clients for seeding and assertions

No logrus, no cobra, no viper. `slog` for logging, `flag` for any test-binary
flags, `errors.Join` for aggregating cleanup errors.

## Interface / API

### Top-level entry point

```go
package libtftest

// TestCase is the primary handle returned from New. It owns a LocalStack
// container (or a reference to a shared one), a scratch workspace, and the
// AWS SDK config used for seeding and assertions.
type TestCase struct {
    t       testing.TB
    stack   *localstack.Container
    work    *tf.Workspace
    awsCfg  aws.Config
    prefix  string              // unique per test, e.g. "ltt-a7f2k9"
    vars    map[string]any
    opts    Options
}

// Options configure a TestCase. Zero-value Options is valid and produces a
// community-edition container with the default service set.
type Options struct {
    // Edition selects Community or Pro. Default: Auto (Pro if
    // LOCALSTACK_AUTH_TOKEN is set, otherwise Community).
    Edition localstack.Edition

    // Services overrides the SERVICES env var. Empty means "all the services
    // the edition supports" (LocalStack's default).
    Services []string

    // Image overrides the container image. Defaults to the OSS image
    // ("localstack/localstack:latest"). Set this to use Pro
    // ("localstack/localstack-pro:latest"), an airgapped mirror, or a
    // custom image. Also configurable via LIBTFTEST_LOCALSTACK_IMAGE env var.
    Image string

    // ModuleDir is the path to the Terraform module under test (required).
    ModuleDir string

    // Vars is passed through to terraform.Options.Vars. For vars that depend
    // on tc.Prefix() (which is only available after New returns), use SetVar.
    Vars map[string]any

    // Reuse attaches to a shared container started via harness.TestMain
    // instead of creating a new one. In per-package mode (the common case),
    // New auto-detects the shared container via a package-level var set by
    // harness.Run — you do not need to set Reuse explicitly. Use Reuse only
    // for advanced cases: a custom TestMain, cross-package sharing, or
    // attaching to an externally-started container. If both Reuse and the
    // auto-detected container are present, Reuse takes precedence.
    Reuse *localstack.Container

    // PersistOnFailure keeps the container and scratch dir alive on failure
    // for post-mortem debugging. Honors LIBTFTEST_PERSIST_ON_FAILURE env var.
    PersistOnFailure bool

    // InitHooks are files written into /etc/localstack/init/ready.d/ before
    // the container is considered healthy. See "Init hooks" below.
    InitHooks []localstack.InitHook

    // AutoPrefixVars, when true, auto-injects tc.Prefix() into any
    // Terraform variable named "name_prefix" if it exists in the module.
    // Default false — explicit SetVar is the primary path.
    AutoPrefixVars bool

    // EdgeURLOverride routes all AWS SDK and Terraform traffic to a custom
    // endpoint instead of the LocalStack container's mapped edge port. The
    // primary use case is sneakystack: start LocalStack as usual, start
    // sneakystack pointed at it, and set EdgeURLOverride to the sneakystack
    // URL so gap services (IAM Identity Center, Organizations, etc.) are
    // handled by sneakystack while everything else passes through to
    // LocalStack. Empty means "use the container's own edge URL".
    EdgeURLOverride string
}

// New creates a TestCase. It starts LocalStack (or attaches to a shared one),
// copies the module into a scratch workspace, writes the provider override,
// and registers cleanup with t.Cleanup. It calls t.Fatal on any setup error.
func New(t testing.TB, opts Options) *TestCase

// SetVar sets or overrides a single Terraform variable. Use this for vars
// that depend on tc.Prefix(), which is only available after New returns.
func (tc *TestCase) SetVar(key string, val any)

// Apply runs `terraform init` + `terraform apply -auto-approve` and returns
// the Terratest options so callers can chain additional operations.
func (tc *TestCase) Apply() *terraform.Options

// ApplyE is the error-returning variant for negative tests.
func (tc *TestCase) ApplyE() (*terraform.Options, error)

// Plan runs `terraform init` + `terraform plan -out` and returns a PlanResult
// containing the parsed plan JSON. Use for golden-file testing or asserting
// on planned changes without a full apply cycle.
func (tc *TestCase) Plan() *PlanResult

// PlanE is the error-returning variant.
func (tc *TestCase) PlanE() (*PlanResult, error)

// PlanResult holds the output of a terraform plan.
type PlanResult struct {
    JSON     []byte             // raw `terraform show -json` output
    FilePath string             // path to the binary plan file
    Changes  PlanChanges        // parsed summary of resource changes
}

// PlanChanges summarizes the resource-level diff from a plan.
type PlanChanges struct {
    Add     int                // resources to create
    Change  int                // resources to update in-place
    Destroy int                // resources to destroy
}

// Output reads a single Terraform output. Shortcut for terraform.Output.
func (tc *TestCase) Output(name string) string

// AWS returns a cached aws.Config pointed at the LocalStack container.
// All clients built from this config will hit the container's edge port.
func (tc *TestCase) AWS() aws.Config

// Prefix returns a unique string that tests should embed in any resource
// name (bucket, table, role, etc.) to guarantee parallel safety.
func (tc *TestCase) Prefix() string
```

### Usage — the 10-line happy path

```go
func TestS3Module_EncryptionEnforced(t *testing.T) {
    t.Parallel()

    tc := libtftest.New(t, libtftest.Options{
        ModuleDir: "../../modules/s3-bucket",
    })
    tc.SetVar("bucket_name", tc.Prefix()+"-logs")

    tc.Apply()

    bucket := tc.Output("bucket_id")
    assert.S3.BucketHasEncryption(t, tc.AWS(), bucket, "AES256")
    assert.S3.BucketBlocksPublicAccess(t, tc.AWS(), bucket)
}
```

Everything else — container startup, provider override, backend override,
`terraform destroy`, container teardown — is handled by `t.Cleanup` callbacks
registered inside `libtftest.New`.

### Provider override

`tf/override.go` writes two override files into the scratch workspace:

**`_libtftest_override.tf.json`** — provider endpoints (port is dynamic, shown
here as `<EDGE_PORT>` — substituted at render time from the testcontainers
mapped port, which may not be 4566 when multiple containers run in parallel):

```json
{
  "provider": {
    "aws": {
      "region": "us-east-1",
      "access_key": "test",
      "secret_key": "test",
      "skip_credentials_validation": true,
      "skip_metadata_api_check": true,
      "skip_requested_account_id": true,
      "s3_use_path_style": true,
      "endpoints": {
        "s3": "http://localhost:<EDGE_PORT>",
        "dynamodb": "http://localhost:<EDGE_PORT>",
        "iam": "http://localhost:<EDGE_PORT>",
        "sts": "http://localhost:<EDGE_PORT>",
        "ssm": "http://localhost:<EDGE_PORT>",
        "secretsmanager": "http://localhost:<EDGE_PORT>",
        "sqs": "http://localhost:<EDGE_PORT>",
        "sns": "http://localhost:<EDGE_PORT>",
        "lambda": "http://localhost:<EDGE_PORT>",
        "kms": "http://localhost:<EDGE_PORT>",
        "cloudwatch": "http://localhost:<EDGE_PORT>",
        "logs": "http://localhost:<EDGE_PORT>",
        "events": "http://localhost:<EDGE_PORT>",
        "kinesis": "http://localhost:<EDGE_PORT>",
        "firehose": "http://localhost:<EDGE_PORT>"
      }
    }
  }
}
```

**`_libtftest_backend_override.tf.json`** — forces local backend:

```json
{
  "terraform": {
    "backend": {
      "local": {}
    }
  }
}
```

This prevents a module that declares `backend "s3"` from accidentally hitting a
real S3 backend during tests. Terraform state is stored in the scratch workspace
under `t.TempDir()`.

The JSON form is deliberate: we do not have to parse the user's existing
provider block, and Terraform merges overrides key-by-key.

### AWS clients

```go
package awsx

// New returns an aws.Config whose BaseEndpoint routes every service
// to the given LocalStack edge URL. Credentials are the LocalStack dummies.
func New(ctx context.Context, edgeURL string) (aws.Config, error) {
    return config.LoadDefaultConfig(ctx,
        config.WithBaseEndpoint(edgeURL),
        config.WithRegion("us-east-1"),
        config.WithCredentialsProvider(
            credentials.NewStaticCredentialsProvider("test", "test", ""),
        ),
    )
}

// Typed constructors for the common services — thin wrappers that add the
// s3.UsePathStyle option, DynamoDB disable-ssl flag, etc.
func NewS3(cfg aws.Config) *s3.Client
func NewDynamoDB(cfg aws.Config) *dynamodb.Client
func NewIAM(cfg aws.Config) *iam.Client
func NewSSM(cfg aws.Config) *ssm.Client
func NewSecrets(cfg aws.Config) *secretsmanager.Client
func NewSQS(cfg aws.Config) *sqs.Client
func NewSNS(cfg aws.Config) *sns.Client
func NewLambda(cfg aws.Config) *lambda.Client
func NewKMS(cfg aws.Config) *kms.Client
func NewKinesis(cfg aws.Config) *kinesis.Client
func NewSTS(cfg aws.Config) *sts.Client
```

We use SDK v2's `config.WithBaseEndpoint` to route all services to the
LocalStack edge URL. This is the current recommended pattern — the older
`EndpointResolverV2` and `WithEndpointResolver` approaches are deprecated.
Callers never have to configure endpoints themselves.

### Fixtures (pre-apply seeding)

Some modules expect resources to pre-exist (e.g. an SSM parameter holding a
shared KMS key ARN). Fixtures let tests seed data before `terraform apply`
without dropping into raw SDK calls.

```go
package fixtures

func SeedS3Object(t testing.TB, cfg aws.Config, bucket, key string, body []byte)
func SeedDynamoItem(t testing.TB, cfg aws.Config, table string, item map[string]types.AttributeValue)
func SeedSSMParameter(t testing.TB, cfg aws.Config, name, value string, secure bool)
func SeedSecret(t testing.TB, cfg aws.Config, name string, value string)
func SeedSQSMessage(t testing.TB, cfg aws.Config, queueURL, body string)
```

Each Seed\* fn registers a `t.Cleanup` that removes the fixture, so parallel
tests don't leak state into each other via a reused container.

### Assertions (post-apply checks)

The `assert` sub-package exposes domain helpers keyed to the resources the
module under test typically creates. Assertions take `testing.TB` and fail via
`t.Errorf` / `t.Fatalf` so they compose with the standard testing framework.

```go
package assert

var S3       s3Asserts
var IAM      iamAsserts
var DynamoDB dynamoAsserts
var Lambda   lambdaAsserts
var SSM      ssmAsserts

// S3 examples:
func (s3Asserts) BucketExists(t testing.TB, cfg aws.Config, name string)
func (s3Asserts) BucketHasEncryption(t testing.TB, cfg aws.Config, name, algo string)
func (s3Asserts) BucketHasVersioning(t testing.TB, cfg aws.Config, name string)
func (s3Asserts) BucketBlocksPublicAccess(t testing.TB, cfg aws.Config, name string)
func (s3Asserts) BucketHasTag(t testing.TB, cfg aws.Config, name, key, want string)

// IAM examples (Pro-only, enforced IAM):
func (iamAsserts) RoleExists(t testing.TB, cfg aws.Config, name string)
func (iamAsserts) RoleHasInlinePolicy(t testing.TB, cfg aws.Config, role, policy string)
func (iamAsserts) PolicyDocumentMatches(t testing.TB, cfg aws.Config, arn string, want iam.PolicyDocument)
```

The `var S3 s3Asserts` pattern uses zero-size structs as method namespaces. This
gives IDE-friendly grouping (`assert.S3.BucketExists(...)`) that reads more
naturally than flat functions (`assert.S3BucketExists(...)`) and avoids
polluting the package namespace as the assertion catalog grows. Each unexported
type prevents external construction — callers use the package-level vars.

Assertions dealing with Pro-only features (real IAM enforcement, EKS, full RDS)
are declared in files behind a `//go:build localstack_pro` tag _or_ call
`libtftest.RequirePro(t)` at the top — see "Edition gating" below.

### Container lifecycle modes

Three modes, selected at test-binary level:

| Mode                   | Where it starts                                     | When to use                                                                   |
| ---------------------- | --------------------------------------------------- | ----------------------------------------------------------------------------- |
| **Per-test** (default) | `libtftest.New`                                     | Max isolation, small modules, CI matrix jobs                                  |
| **Per-package**        | `harness.TestMain`                                  | Typical case: one container shared across the `Test*` funcs in a package      |
| **Per-suite**          | `harness.SharedMain` with `LIBTFTEST_CONTAINER_URL` | Local dev loop; `docker run` LS yourself, export the URL, all packages attach |

Per-package mode is the one most teams will adopt. `harness.TestMain` is a
copy-pasteable `TestMain`:

```go
func TestMain(m *testing.M) {
    harness.Run(m, harness.Config{
        Edition: localstack.EditionAuto,
    })
}
```

Under the hood, `harness.Run` starts one container, stores a reference in
`harness.shared` (a package-level `*localstack.Container` guarded by
`sync.Once`), runs the tests, and stops the container. `libtftest.New` calls
`harness.Current()` to check for a shared container — if non-nil, it reuses that
container instead of starting a new one. Explicit `Reuse` is not needed for this
common path — see the `Reuse` field doc for advanced use cases.

### Edition gating

```go
// RequirePro skips the test when the running container is Community edition.
// Detection is automatic: it queries the container's /_localstack/health
// endpoint and checks the edition field — no env-var guessing required.
func RequirePro(t testing.TB)

// RequireServices skips the test when any of the named services is not
// available in the running container's edition.
func RequireServices(t testing.TB, services ...string)
```

The `services.available` field on `GET /_localstack/health` is the source of
truth; we cache the response on the TestCase so repeated calls are free.

Pro-only assertion methods (e.g. `assert.IAM.RoleHasInlinePolicy`) call
`RequirePro(t)` internally, so callers don't need to remember which assertions
require Pro — the test auto-skips with a clear message if the running container
is Community edition.

## Implementation

### LocalStack container startup

```go
package localstack

type Container struct {
    ID       string
    EdgeURL  string            // e.g. http://localhost:49314
    Edition  Edition
    Services map[string]string // name -> status from /health
    ctr      testcontainers.Container
}

func Start(ctx context.Context, cfg Config) (*Container, error) {
    ctr, err := testcontainers.Run(ctx, cfg.Image(),
        testcontainers.WithExposedPorts("4566/tcp"),
        testcontainers.WithEnv(cfg.Env()),
        testcontainers.WithWaitStrategy(
            wait.ForHTTP("/_localstack/health").
                WithPort("4566/tcp").
                WithStartupTimeout(90 * time.Second).
                WithResponseMatcher(allServicesReady),
        ),
        testcontainers.WithHostConfigModifier(func(hc *container.HostConfig) {
            hc.AutoRemove = true
            hc.Mounts = cfg.Mounts()                  // init hooks dir
        }),
    )
    // ...
}
```

Key details:

1. **Docker socket detection** — `internal/dockerx` pings the daemon before we
   call testcontainers. On failure we print a remediation message
   (`colima start`, `rancher-desktop`, etc.) and `t.Fatal`. Donald runs Lima
   with nested KVM — the default `DOCKER_HOST` works, but we honor
   `TESTCONTAINERS_HOST_OVERRIDE` for non-obvious setups.
2. **Auth token passthrough** — `LOCALSTACK_AUTH_TOKEN` is forwarded to the
   container when present. If missing and `Edition == EditionPro`, startup fails
   fast with a clear error.
3. **Health check** — we do not trust "container running" as "ready". The waiter
   polls `/_localstack/health` and blocks until every declared service is
   `running` or `available`. `allServicesReady` is a `ResponseMatcher` that
   json-decodes the body and returns true only when no service is in state
   `initializing` or `error`.
4. **Ryuk** — testcontainers' reaper is enabled by default. If a test panics,
   Ryuk cleans up the container. We additionally register `t.Cleanup` so normal
   paths don't wait for Ryuk's TTL.
5. **Port mapping** — the edge port is _mapped_, not fixed. `EdgeURL` is
   computed from `container.Endpoint(ctx, "")` and threaded through the provider
   override so parallel containers never collide.

### Workspace copy

`tf.Workspace` owns a `t.TempDir()`-rooted scratch copy of the module. Copying
(instead of writing into the real source) has three benefits:

- The real `.tf` files stay pristine — no accidental commits of override files.
- `.terraform/` caches are per-workspace, so parallel tests don't race on plugin
  lock files.
- We can freely mutate the workspace (inject overrides, write backend config)
  without touching the module source.

```go
func NewWorkspace(t testing.TB, moduleDir string) *Workspace {
    dst := t.TempDir()
    if err := copyTree(moduleDir, dst); err != nil { t.Fatal(err) }
    return &Workspace{Dir: dst, src: moduleDir}
}
```

`copyTree` is a plain `filepath.WalkDir` using `io.Copy` — no dependency on
`cp`, no shell out. Symlinks are followed once (to catch
`modules/shared -> ../shared` patterns) and then rejected to avoid cycles.

### Provider override rendering

Given the container's `EdgeURL`, we render the JSON override from a template
backed by the LocalStack service catalog. The catalog lives in
`localstack/services.go` as a Go slice, generated from LocalStack's
`localstack.services` package upstream via `go generate`. This keeps the
endpoint list in sync with upstream without a runtime dependency.

A second override file (`_libtftest_backend_override.tf.json`) forces
`backend "local"` so modules that declare a remote backend don't accidentally
reach production state stores during tests.

### terraform.Options construction

```go
func (tc *TestCase) buildOptions() *terraform.Options {
    return terraform.WithDefaultRetryableErrors(tc.t, &terraform.Options{
        TerraformDir: tc.work.Dir,
        Vars:         tc.vars,
        EnvVars: map[string]string{
            "AWS_ACCESS_KEY_ID":     "test",
            "AWS_SECRET_ACCESS_KEY": "test",
            "AWS_DEFAULT_REGION":    "us-east-1",
            "TF_PLUGIN_CACHE_DIR":   pluginCacheDir(),
            "TF_IN_AUTOMATION":      "1",
        },
        NoColor:      true,
        Lock:         true,
        LockTimeout:  "60s",
        Logger:       logger.Discard, // we handle logging ourselves
        PlanFilePath: filepath.Join(tc.work.Dir, "libtftest.plan"),
    })
}
```

`pluginCacheDir()` returns `$XDG_CACHE_HOME/libtftest/plugin-cache` (or the
macOS `~/Library/Caches` equivalent), created on first call. This shaves 5–15
seconds off every `terraform init` after the first. Note: Terraform's plugin
cache does not support concurrent writes. When multiple packages run in parallel
(`go test ./...`), each invocation races on the cache directory. Terraform
handles this internally with file locks (since 1.4+); libtftest does not add
additional locking.

### Parallel-safe naming

```go
package naming

// Prefix returns a 10-char lowercase string: "ltt-" + 6 hex chars of
// hash(test name + pid + nanotime). Deterministic within a test run so
// failures are reproducible; unique across parallel runs.
func Prefix(t testing.TB) string
```

Tests are expected to interpolate `tc.Prefix()` into every resource name. The
harness does _not_ mutate user Vars automatically — that would be too magic and
would collide with user-chosen naming schemes.

### Init hooks

LocalStack supports scripts in `/etc/localstack/init/ready.d/` that run once the
container is healthy. `localstack.InitHook` lets callers seed state the hard way
(e.g. create a CA cert in ACM before terraform runs):

```go
type InitHook struct {
    Name   string      // becomes the filename
    Script string      // bash content, runs inside the container
}
```

Hooks are passed via `Options.InitHooks` — no functional-option constructors.

Hooks are written to a temp dir, bind-mounted at
`/etc/localstack/init/ready.d/`, and their completion is observed via
`/_localstack/health` (the `init` field reports per-hook state).

### Cleanup ordering

`t.Cleanup` calls run LIFO. We register callbacks in this order inside `New` so
they execute in the correct reverse order:

| Registration order | Callback                       | Execution order (LIFO) |
| ------------------ | ------------------------------ | ---------------------- |
| 1st (earliest)     | Stop + remove container        | Runs **last**          |
| 2nd                | Destroy terraform state        | Runs **second**        |
| 3rd (latest)       | Close AWS clients / flush logs | Runs **first**         |

Execution sequence: logs flush first (so we still have clients when dumping
state on failure), then `terraform destroy` runs against a still-live container,
then the container dies. `PersistOnFailure` short-circuits the destroy and
container-stop callbacks when `t.Failed()` is true so a human can `docker exec`
into the wreckage.

## sneakystack

LocalStack — even Pro — has blind spots. IAM Identity Center permission set
provisioning, Organizations account lifecycle, Control Tower, Resource Explorer,
and a handful of other control-plane APIs either don't exist or exist as no-op
stubs. These are exactly the APIs Donald's platform team touches most, so "just
skip those tests" is not acceptable.

The `sneakystack` package (`sneakystack/`) fills these gaps. Architecturally
it's a Go HTTP proxy that sits in front of LocalStack: requests for services
sneakystack implements (permission sets, org accounts, etc.) are handled locally
against an in-memory store; everything else is transparently forwarded to
LocalStack. From the AWS SDK's perspective there is a single endpoint, and that
endpoint speaks the real AWS wire protocol for every service — real, emulated,
or faked.

A standalone binary (`cmd/sneakystack/`) packages the proxy into a Docker
container for use outside of Go tests — e.g., manual `docker-compose` setups or
non-Go test frameworks that need the same gap-filling.

### Store interface

sneakystack's persistence is abstracted behind a `Store` interface. The v1
implementation is plain Go maps protected by `sync.RWMutex` — no external
dependencies. If query complexity or concurrency demands grow, the interface
allows swapping in a heavier backend (e.g., `modernc.org/sqlite`) without
touching service handlers.

```go
package sneakystack

// Store is the persistence abstraction for service handlers. Each service
// gets its own typed wrapper built on top of this.
type Store interface {
    Put(ctx context.Context, kind, id string, obj any) error
    Get(ctx context.Context, kind, id string) (any, error)
    List(ctx context.Context, kind string, filter Filter) ([]any, error)
    Delete(ctx context.Context, kind, id string) error
}

type Filter struct {
    Parent string            // e.g. instance ARN, OU id
    Tags   map[string]string // optional tag match
}
```

Service handlers wrap `Store` with typed accessors:

```go
func (s *SSOAdminService) GetPermissionSet(ctx context.Context, instanceARN, id string) (*PermissionSet, error) {
    obj, err := s.store.Get(ctx, "permission-set", id)
    // type assert, validate parent, return
}
```

### Sidecar interface

The `harness` package defines a `Sidecar` interface that sneakystack (and any
future auxiliary services) implements:

```go
package harness

// Sidecar is implemented by packages that provide auxiliary services
// (in-process or containerized) that sit between libtftest and LocalStack.
type Sidecar interface {
    // Start launches the sidecar with the given LocalStack edge URL as
    // its downstream target. Returns the URL callers should use instead
    // of the raw LocalStack URL.
    Start(ctx context.Context, localstackURL string) (edgeURL string, err error)

    // Stop shuts down the sidecar.
    Stop(ctx context.Context) error

    // Healthy returns true when the sidecar is ready to accept traffic.
    Healthy(ctx context.Context) bool
}
```

### Integration with the harness

libtftest treats sneakystack as opt-in. The integration surface is small:

```go
func TestMain(m *testing.M) {
    harness.Run(m, harness.Config{
        Edition: localstack.EditionAuto,
        Sidecars: []harness.Sidecar{
            sneakystack.NewSidecar(sneakystack.Config{
                Services: []string{"sso-admin", "organizations"},
            }),
        },
    })
}
```

Under the hood, `harness.Run` starts the LocalStack container first, passes its
edge URL to sneakystack as a downstream target, starts sneakystack as an
in-process goroutine, waits for both to be healthy, and then exposes the
sneakystack URL as the effective edge URL. `libtftest.New` sees
`EdgeURLOverride` set and plumbs it through the provider override and `awsx.New`
so every SDK client and every Terraform resource in the module under test hits
sneakystack first.

sneakystack can also run standalone via its Docker container for non-Go
consumers. In that mode, users point `LIBTFTEST_CONTAINER_URL` at the
sneakystack address directly.

The package coupling is intentionally loose: libtftest knows sneakystack as "a
URL that speaks AWS", nothing more. sneakystack knows LocalStack as "a URL I can
forward to", nothing more. Either can be replaced independently. The expected
failure mode — a request hits sneakystack for a service it doesn't proxy and
sneakystack forwards blindly to LocalStack which returns an unhelpful error — is
handled within sneakystack's request routing, not in libtftest.

### Error classification

Errors are classified at the boundary so callers get actionable messages instead
of raw stack traces.

| Failure                  | Detection                                       | Action                                                                          |
| ------------------------ | ----------------------------------------------- | ------------------------------------------------------------------------------- |
| Docker daemon down       | `dockerx.Ping` fails                            | `t.Fatalf` with `colima start` / `rancher-desktop` hint                         |
| Image pull rate-limited  | testcontainers error contains "toomanyrequests" | Suggest `docker login` and GH Actions `docker/login-action`                     |
| Port exhaustion          | `bind: address already in use`                  | Retry with a new ephemeral port; fail after 3 attempts                          |
| Auth token missing (Pro) | `Edition == Pro` and env empty                  | `t.Fatal` with link to wiki page on token acquisition                           |
| Health check timeout     | `WaitingFor` returns timeout                    | Dump `docker logs` to test artifact, `t.Fatal`                                  |
| Service not ready        | `/_localstack/health` reports `error`           | Dump `docker logs`, name the offending service, `t.Fatal`                       |
| Terraform init fails     | terratest returns error                         | Dump `.terraform.lock.hcl` + provider override, `t.Fatal`                       |
| Terraform apply fails    | terratest returns error                         | Dump plan file + override + LS logs, run destroy, `t.Fatal`                     |
| Destroy fails            | terratest returns error                         | Log as warning; remaining resources are destroyed when the container is removed |
| Cleanup panics           | `recover()` in cleanup                          | `errors.Join` with other cleanup errors, report via `t.Error`                   |

Cleanup errors are aggregated with `errors.Join` so a failure in one cleanup
step does not mask failures in the others. The joined error is logged via
`t.Errorf` (not `Fatalf`) so the primary test result is preserved.

All dumps land in a per-test `t.TempDir()` subdirectory plus, on CI, the
`$LIBTFTEST_ARTIFACT_DIR` path (e.g. `$GITHUB_STEP_SUMMARY`'s sibling) so GitHub
Actions' upload-artifact can pick them up.

## Testing Strategy

### Tests of libtftest itself

Three layers:

1. **Unit tests** — Logic that does not need a container: naming, override
   rendering, edition detection, cleanup ordering. These run in `go test ./...`
   with no external dependencies and target <2s total.
2. **Integration tests** — Spin up a real Community LocalStack container and
   exercise the full lifecycle against a tiny fixture module under
   `testdata/mod-s3/`. Marked with `//go:build integration` and run in a
   separate CI job.
3. **Pro integration tests** — Same as above but with Pro image and
   `LOCALSTACK_AUTH_TOKEN`. Behind `//go:build integration && localstack_pro`.
   Runs only on the `main` branch with the token in GH Actions secrets.

### Tests of consumer modules (the primary use case)

Consumers write one test file per module under `test/` in their module repo. We
publish an example:

```
terraform-aws-s3-bucket/
├── main.tf
├── variables.tf
├── outputs.tf
└── test/
    ├── go.mod
    ├── main_test.go        # TestMain with harness.Run
    ├── s3_bucket_test.go   # table-driven tests
    └── testdata/
```

CI runs `go test -tags=integration ./test/...` in a single GH Actions job per
module. Typical wall-clock for a 5-test package with shared container mode:
45–75 seconds including container start.

### Matrix coverage

Nightly matrix covers the last three supported Terraform minor versions
(minimum: oldest still receiving HashiCorp security patches), the last two
LocalStack minor versions plus `latest`, crossed with both editions. Results
published to a dashboard (Backstage TechDocs site) so regressions are visible
without trawling Actions logs.

### Golden-file plan testing

The `tc.Plan()` method runs `terraform plan -out` and returns a `PlanResult`
containing the parsed plan JSON. Tests can assert on plan JSON directly or diff
against a checked-in golden file. This catches "the module unexpectedly wants to
destroy the VPC" bugs without requiring a full apply cycle — useful for modules
whose real apply takes too long even on LocalStack.

## Useful Extras (candidates for v1.1+)

Ideas that fall out of this design naturally and are worth flagging even if they
don't ship in v1:

- **`chaos/`** — thin wrapper around LocalStack's Chaos API (Pro) to inject
  regional outages and latency mid-test. Lets module authors verify retry logic
  without mocking.
- **`snapshot/`** — wraps LocalStack PODs (Pro) so a test can snapshot state,
  mutate it, and restore between sub-tests. Speeds up table-driven tests
  dramatically when the setup cost dominates.
- **`policy/`** — hook point for Conftest/OPA policy evaluation against the plan
  JSON. Composes with existing Wiz custom-framework tooling: policies authored
  once, run both at test time and at scan time.
- **`cost/`** — parses the plan for resource counts and emits a "resources
  created" metric. Zero-dollar on LocalStack, but the same hook feeds Infracost
  on real-AWS runs later.
- **`drift/`** — runs `terraform plan` after `apply` and fails if non-empty.
  Catches non-idempotent modules.
- **`harness.TFExec`** — an alternative runner using HashiCorp's `tfexec`
  instead of terratest's shell-out. Faster (no subprocess per op), gives
  structured plan output. We'd expose both and let callers choose.
- **Trivy / Checkov integration** — run static scanners over the scratch
  workspace after override injection, so the security scan sees the same code
  the apply does.
- **Init-hook cookbook** — a curated `hooks/` sub-package with pre-built hooks
  for common needs: "pre-create a KMS key", "seed an ACM cert", "populate
  Route53 hosted zone".

None of these belong in v1, but the package layout above leaves room for each to
land as an additive sub-package without breaking the core API.

## Decisions

Resolved from the original open questions during design review.

1. **Shared container by default.** Per-package (shared) is the default mode
   when `harness.Run` is used in `TestMain`. The harness emits a loud warning if
   a test forgets to namespace resources with `tc.Prefix()`. Per-test mode (max
   isolation) is the default when no `TestMain` calls `harness.Run` — each
   `libtftest.New` starts its own container.

2. **Opt-in auto-prefix for Vars.** `Options.AutoPrefixVars bool` (default
   false). When enabled, the harness auto-injects `tc.Prefix()` into any
   Terraform variable named `name_prefix` if it exists in the module's
   variables. Explicit `SetVar` remains the primary path.

3. **Terratest first, tfexec in v1.1.** Ship with Terratest as the runner — it's
   the team's muscle memory and has the broadest adoption. Add a `tfexec` runner
   in v1.1 behind the same `*terraform.Options` facade for callers who want
   structured output and no subprocess overhead.

4. **Image is fully configurable with an OSS default.** Default image is
   `localstack/localstack:latest` (Community/OSS). Callers override via
   `Options.Image` or the `LIBTFTEST_LOCALSTACK_IMAGE` env var to use Pro,
   airgapped mirrors, or custom images. The nightly CI matrix runs against
   `latest` to catch upstream regressions early.

5. **Monorepo for v1.** The `assert` sub-package stays in this module. Split
   only if the assertion catalog exceeds ~2k LOC and its release cadence
   diverges meaningfully from the core API.

6. **Runtime skip with auto-detection.** Pro-only assertions call
   `RequirePro(t)` internally — no build tags required. `RequirePro` queries the
   running container's `/_localstack/health` endpoint to detect the edition, so
   it works regardless of how the container was started. Tests auto-skip with a
   clear message on Community edition.

## Claude Code Automation

Repeatable tasks in this repo should be codified as Claude Code skills and
agents to ensure consistency and reduce onboarding friction. Automation lives in
two places:

### Local skills (this repo, `.claude/`)

Skills scoped to this repository for common development workflows:

| Skill                     | Trigger                                   | What it does                                                                                                                                                                              |
| ------------------------- | ----------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `add-assertion`           | Adding a new assertion type               | Scaffolds a new file in `assert/`, creates the typed struct, adds the package-level var, generates test stubs with table-driven patterns. Prompts for service name and assertion methods. |
| `add-fixture`             | Adding a new fixture function             | Scaffolds a `Seed*` function in `fixtures/` with the matching `t.Cleanup` teardown and test stubs.                                                                                        |
| `add-sneakystack-service` | Adding a new gap service                  | Scaffolds a new service handler in `sneakystack/services/`, registers it in the proxy router, creates the Store typed wrapper, and generates test stubs.                                  |
| `add-awsx-client`         | Adding a new AWS SDK client               | Scaffolds a typed constructor in `awsx/` with the correct endpoint resolver configuration and test stubs.                                                                                 |
| `scaffold-module-test`    | Bootstrapping tests for a consumer module | Generates a `test/` directory in a Terraform module repo with `go.mod`, `TestMain`, and a starter test file wired to libtftest.                                                           |

Each skill should follow the project's conventions: stdlib-first, Uber Go Style
Guide, table-driven tests, and `slog` for logging.

### Plugin skills (donaldgifford/claude-skills, `infrastructure-as-code` plugin)

The `infrastructure-as-code` plugin at
<https://github.com/donaldgifford/claude-skills> needs new skills and agents for
libtftest-aware Terraform module testing:

| Component         | Type  | Description                                                                                                                                                                                                                                                                          |
| ----------------- | ----- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `tftest`          | skill | Guidance for writing Terraform module integration tests using libtftest. Covers TestCase API, fixture/assertion patterns, container lifecycle modes, and sneakystack integration. Triggers when working in `test/` dirs of Terraform modules or when `libtftest` appears in imports. |
| `tftest:scaffold` | skill | Scaffold a complete test directory for a Terraform module: `go.mod` with libtftest dependency, `TestMain` with `harness.Run`, starter test file with table-driven pattern, and `.github/workflows` integration.                                                                      |
| `tftest:add-test` | skill | Add a new test case to an existing libtftest test file. Prompts for the resource under test and generates assertions using the appropriate `assert.*` helpers.                                                                                                                       |
| `tftest-reviewer` | agent | Reviews libtftest test code for best practices: proper use of `tc.Prefix()` for parallel safety, correct cleanup ordering, appropriate edition gating for Pro-only assertions, fixture teardown, and coverage of key module outputs.                                                 |

These should be implemented during the Week 5-6 rollout phase alongside the
reusable GH Actions workflow, so module teams have both CI automation and Claude
Code assistance from day one.

## Rollout

1. **Week 1-2** — Scaffold module, port the container lifecycle and provider
   override from an existing spike, ship unit tests.
2. **Week 3** — `awsx`, `fixtures`, `assert` for S3 + DynamoDB + IAM + SSM +
   Secrets Manager. One example module consumer. Scaffold `sneakystack` package
   with Store interface and SSO Admin + Organizations handlers.
3. **Week 4** — `harness.TestMain`, plugin cache, artifact dumping on failure.
   Dogfood against `terraform-aws-s3-bucket` and one more module. sneakystack
   Docker container + Sidecar integration.
4. **Week 5** — CI integration: reusable GH Actions workflow
   (`libtftest-module.yml`) that any module repo can `workflow_call` into.
   Publish v0.1.0 tag. Implement local `.claude/` skills for this repo
   (`add-assertion`, `add-fixture`, `add-sneakystack-service`,
   `add-awsx-client`).
5. **Week 6** — Add `tftest` skill, `tftest:scaffold`, `tftest:add-test`, and
   `tftest-reviewer` agent to the `infrastructure-as-code` plugin in
   `donaldgifford/claude-skills`. Ship `scaffold-module-test` local skill.
6. **Week 7+** — Broader rollout across module repos; weekly office hours for
   onboarding; track adoption via repo-guardian compliance check.

### Versioning

Pre-v1 releases (`v0.x`) may contain breaking changes between minor versions.
The `TestCase` API (New, SetVar, Apply, Plan, Output, AWS, Prefix) is intended
to stabilize first; assertion and fixture APIs may shift as we learn from early
consumers. v1.0 will carry a no-breaking-changes commitment on the core API
under standard Go module compatibility guarantees. The `assert`, `fixtures`, and
`sneakystack` packages version with the module — no separate release cadence for
v1.

## Related

- `sneakystack/` — LocalStack gap-filling proxy, lives in this module. See the
  "sneakystack" section above.
- [ADR — planned] — Adopt LocalStack Pro across the module-testing fleet.
- [RFC — planned] — Terraform module testing standards.
- Upstream: <https://docs.localstack.cloud/user-guide/integrations/terraform/>
- Upstream: <https://terratest.gruntwork.io/>
