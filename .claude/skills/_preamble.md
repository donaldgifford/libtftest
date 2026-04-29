# Local Skill Preamble

Reference snippet shared by every `libtftest:*` skill in `.claude/skills/`.
Skills should include or restate the relevant pieces in their own `SKILL.md`
body — `_preamble.md` itself is not auto-loaded.

## Project conventions

- **Module path**: `github.com/donaldgifford/libtftest`
- **Go version**: 1.26 (see `go.mod`, pinned by testcontainers-go toolchain)
- **Style guide**: Uber Go Style Guide; enforced by `golangci-lint` v2 with the
  config in `.golangci.yml` (line length 150 via `golines`)
- **Stdlib-first**: `slog` for logging, `errors.Join` for cleanup aggregation.
  No logrus, cobra, viper. Vendor a third-party library only when stdlib really
  doesn't cover it.
- **Tests**: table-driven where the function has multiple input variations.
  Use `t.Parallel()` when safe. Integration tests live behind
  `//go:build integration`.
- **Naming**:
  - `tb testing.TB` (not `t`) for any helper that takes a test handle —
    enforced by the `thelper` linter.
  - `Seed*` for fixture functions in `fixtures/`.
  - Zero-size struct + package-level var pattern for assertion namespaces
    (see `assert/s3.go`).
- **Comments**: comments on exported symbols end with periods (`godot` linter).
  Default to writing no comments at all unless the *why* is non-obvious.
- **`nolint` directives**: must include a specific linter name and an
  explanation, e.g. `//nolint:gosec // path is derived from $HOME, validated`.

## Hard-won gotchas (do not relearn)

- testcontainers-go: `ctr.Endpoint(ctx, "http")` returns the **lowest** numbered
  port. Use `ctr.PortEndpoint(ctx, "4566/tcp", "http")` to get the edge port.
- testcontainers `WithResponseMatcher` is `func(io.Reader) bool`, not
  `func(*http.Response) bool`.
- testcontainers `WithHostConfigModifier` takes
  `func(*container.HostConfig)` from `github.com/moby/moby/api/types/container`.
- AWS SDK Go v2: use `config.WithBaseEndpoint`. **Do not** implement a custom
  `EndpointResolverV2` — deprecated.
- `aws.Config` is 696 bytes. The `gocritic.hugeParam` threshold is raised to
  800 in `.golangci.yml` so the AWS SDK convention of passing it by value
  works.
- terratest v0.56.0 defaults to the `tofu` binary. CI must install
  `hashicorp/setup-terraform@v3` or otherwise put `terraform` on PATH.
- terratest `terraform.Options.PlanFilePath` causes Apply to apply a plan file.
  Use `tf.BuildOptions` (no plan) for Apply, `tf.BuildPlanOptions` (with plan)
  for Plan.
- LocalStack `:latest` requires a Pro auth token. Pin to
  `localstack/localstack:4.4`.
- `t.Setenv` conflicts with `t.Parallel()` — pick one.
- `gosec G703` on env-derived paths (`HOME`, `XDG_CACHE_HOME`): annotate the
  `os.MkdirAll` / `os.Stat` line with `//nolint:gosec`, not the `os.Getenv`
  line.

## File layout cheat sheet

```text
libtftest/
├── libtftest.go          # TestCase, New, SetVar, Apply, Plan, Output, AWS, Prefix, RequirePro
├── awsx/                 # AWS SDK v2 client constructors
├── assert/               # Post-apply assertions (zero-size struct + package var)
├── fixtures/             # Pre-apply Seed* functions (paired with t.Cleanup)
├── tf/                   # workspace.go, override.go, options.go (BuildOptions / BuildPlanOptions)
├── localstack/           # container.go, edition.go, health.go, init_hooks.go
├── harness/              # Run, Sidecar interface
├── sneakystack/          # store.go, proxy.go, sidecar.go (HTTP gap-filling proxy)
├── internal/{naming,dockerx,logx}/
├── cmd/{libtftest,sneakystack}/
└── testdata/mod-s3/      # Fixture Terraform module used by integration tests
```

## When generating new code

1. Read the closest existing example in the same package before writing.
2. Match the existing function-doc style (`// FuncName does X. Y. Z.`).
3. Add a table-driven test in the matching `_test.go` file.
4. Run `make lint` and `make test-pkg PKG=./<pkg>` before claiming success.
5. Don't introduce a new dependency without a clear reason; check `go.mod`
   first.
