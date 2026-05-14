---
name: libtftest:add-fixture
description: >
  Scaffold a new Seed* fixture function in a per-service package under
  fixtures/<service>/ in libtftest. Use when adding pre-apply data seeding
  for an AWS resource that pairs a PutSomething call with a t.Cleanup
  teardown.
when_to_use: >
  When the user says "add a fixture for X", "we need to seed Y before
  apply", or "the test needs Z to exist already". Activates inside the
  libtftest repo only.
allowed-tools: [Read, Write, Edit, Bash, Grep]
---

# libtftest:add-fixture

Adds (or extends) a per-service fixture package under
`fixtures/<service>/`. As of IMPL-0004 / DESIGN-0003 Part 1 fixtures
follow the same per-service-package layout as `assert/`: one Go package
per AWS service, package name = lowercase service short name, **function
names drop the service prefix** (the package name carries it).
Consumers import with an alias:

```go
import s3fix "github.com/donaldgifford/libtftest/fixtures/s3"

s3fix.SeedObject(t, tc.AWS(), bucket, "k", "v")
s3fix.SeedObjectContext(t, ctx, tc.AWS(), bucket, "k", "v")
```

Every fixture function **must** register a `tb.Cleanup` that removes
whatever it created. The cleanup pairing is the contract — without it,
parallel tests collide.

## Repo conventions

- `tb testing.TB` (not `t`) — the `thelper` linter enforces this
- Fail-fast on seed errors with `tb.Fatalf` (the test cannot proceed
  without the fixture)
- Fail-soft on cleanup errors with `tb.Errorf` (the test result is what
  matters, not whether teardown succeeded — but log it)
- Doc comments end with periods (`godot`)
- Use `awsx.New<Service>(cfg)` for the AWS client. If the constructor is
  missing, run `/libtftest:add-awsx-client` first.
- Pre-apply seeding runs **before** `tc.Apply()`, so the fixture must
  point at LocalStack via the `aws.Config` from `tc.AWS()` — that's what
  the caller supplies as the `cfg` argument.
- **Per-service-package layout (required as of v0.2.0).** Each AWS
  service gets its own `fixtures/<service>/` package. Function names
  drop the service prefix: `s3fix.SeedObject`, not
  `fixtures.SeedS3Object`. The SDK package is imported under an alias
  (`s3sdk "github.com/aws/aws-sdk-go-v2/service/s3"`) to avoid
  collision.
- **Companion `doc.go` (per [INV-0003][inv-0003] / CLAUDE.md).** Every
  package ships a dedicated `doc.go` containing only the `package` line
  and a multi-paragraph godoc comment. No imports, types, or constants
  belong in `doc.go`.
- **Paired-method pattern (required as of v0.2.0).** Every `Seed*`
  function has a `Seed*Context` variant accepting `context.Context`.
  The non-context shim forwards with `tb.Context()`. Cleanup callbacks
  use `context.WithoutCancel(ctx)` so they survive test-end
  cancellation.

## Procedure

1. **Confirm the fixture.** Ask the user for:
   - Service short name (e.g., `dynamodb`, `kinesis`) — this is both
     the package name and the directory name
   - Function name (dropping the service prefix — e.g., `SeedItem`,
     `SeedStream`)
   - Parameters (typed signature, in idiomatic order)
   - The Put/Create call to use
   - The matching Delete/Destroy call for cleanup

2. **Verify `awsx.New<Service>` exists.** Read `awsx/clients.go`. If the
   constructor is missing, halt and recommend:
   > "I need `awsx.New<Service>` first. Run `/libtftest:add-awsx-client`
   > with service=`<svc>`, then come back to this skill."

3. **Check whether `fixtures/<service>/` exists.**
   - If yes: add the new function(s) to the existing `<service>.go`
     file. Do not create a duplicate package.
   - If no: create the package directory with two files:
     - `fixtures/<service>/<service>.go` from
       `references/fixture-template.go.tmpl`
     - `fixtures/<service>/doc.go` — package line + godoc comment only

4. **Generate the function.** Follow the template:
   - Package-level function (no receiver). Drop the service prefix from
     the name — the package carries it.
   - First line: `tb.Helper()`.
   - Use `tb.Fatalf` on the seed error.
   - Bind `cleanupCtx := context.WithoutCancel(ctx)` before registering
     cleanup so destroy survives test-end cancellation.
   - Register cleanup via `tb.Cleanup(func() { ... })`. Use `tb.Errorf`
     (not `Fatalf`) inside the cleanup.
   - Generate both the `*Context` variant and the non-context shim. The
     shim is a one-liner forwarding to `*Context` with `tb.Context()`.
   - Import the AWS SDK service package under the `<service>sdk` alias
     to avoid collision with the package name itself.

5. **Add a unit test** to `fixtures/<service>/<service>_test.go` in the
   `<service>_test` external test package. Use
   `internal/testfake.NewFakeTB()` for unit coverage — verify cancelled
   `ctx` propagation, that the fake records cleanup registration, and
   error reporting. Real-LocalStack exercise belongs behind
   `//go:build integration` in `libtftest_integration_test.go` (or a
   service-specific integration test).

6. **Run lint and test.**
   ```bash
   make lint
   make test-pkg PKG=./fixtures/<service>
   ```
   Common issues:
   - `thelper`: parameter named `t` instead of `tb`
   - `errcheck`: forgot to check return on cleanup call
   - `godot`: missing period at end of doc comment
   - `gosec G104`: explicit error checks needed (the template handles
     this)
   - SDK package collides with the fixtures package name — use the `sdk`
     alias (e.g., `s3sdk`, `ssmsdk`)

## Edge cases

- **Idempotent seed.** Some Put calls fail on existing-resource
  collisions. Either pre-delete (with cleanup pair anyway) or wrap with
  a "if not exists" guard. Document the behavior in the doc comment.
- **Cross-service fixtures.** A test may need to seed both an SSM param
  and an S3 object. Generate them as separate `Seed*` functions in
  their respective packages, not a single multi-service helper. Compose
  at the call site.
- **Force-delete.** Some services don't support immediate deletion
  (Secrets Manager has a 7-30 day recovery window). Use the force-delete
  variant in cleanup (e.g., `ForceDeleteWithoutRecovery: true` for
  Secrets Manager). Match the convention in
  `fixtures/secretsmanager/secretsmanager.go`.
- **Fixture without cleanup.** A few fixtures genuinely consume their
  payload (`sqsfix.SeedMessage` — the test reads and removes it). For
  these, the doc comment must explicitly say "Cleanup: none (consumed by
  test)" and skip the `tb.Cleanup` registration. Rare; ask the user to
  confirm before going this route.

## References

- `references/fixture-template.go.tmpl` — full Seed function template
  with cleanup pair
- `fixtures/s3/s3.go` — Community example (per-service package)
- `fixtures/secretsmanager/secretsmanager.go` — force-delete cleanup
  example
- `fixtures/sqs/sqs.go` — fire-and-forget (no cleanup) example
- `internal/testfake/testfake.go` — shared fake `testing.TB` for unit tests

[inv-0003]: ../../../docs/investigation/0003-package-documentation-convention-and-gomarkdoc-toolchain.md
