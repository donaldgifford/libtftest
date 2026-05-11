---
name: libtftest:add-fixture
description: >
  Scaffold a new Seed* fixture function in libtftest's fixtures/ package.
  Use when adding pre-apply data seeding for an AWS resource that pairs a
  PutSomething call with a t.Cleanup teardown.
when_to_use: >
  When the user says "add a fixture for X", "we need to seed Y before
  apply", or "the test needs Z to exist already". Activates inside the
  libtftest repo only.
allowed-tools: [Read, Write, Edit, Bash, Grep]
---

# libtftest:add-fixture

Adds a `Seed*` function to `fixtures/fixtures.go`. Every fixture function
**must** register a `t.Cleanup` that removes whatever it created. The
cleanup pairing is the contract — without it, parallel tests collide.

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
- **Paired-method pattern (required as of v0.2.0).** Every `Seed*`
  function has a `Seed*Context` variant accepting `context.Context`.
  The non-context shim forwards with `tb.Context()`. Cleanup callbacks
  use `context.WithoutCancel(ctx)` so they survive test-end
  cancellation.

## Procedure

1. **Confirm the fixture.** Ask the user for:
   - Function name (e.g., `SeedDynamoDBItem`, `SeedKinesisStream`)
   - Parameters (typed signature, in idiomatic order)
   - The Put/Create call to use
   - The matching Delete/Destroy call for cleanup

2. **Verify `awsx.New<Service>` exists.** Read `awsx/clients.go`. If the
   constructor is missing, halt and recommend:
   > "I need `awsx.New<Service>` first. Run `/libtftest:add-awsx-client`
   > with service=`<svc>`, then come back to this skill."

3. **Add the function** to `fixtures/fixtures.go` following the template in
   `references/fixture-template.go.tmpl`. Maintain the rough alphabetical
   ordering by service in the file. Add the SDK import to the import block.

4. **Verify the cleanup pair.** The `*Context` function must:
   - Call `tb.Helper()` first
   - Use `tb.Fatalf` on the seed error
   - Bind `cleanupCtx := context.WithoutCancel(ctx)` before registering
     cleanup
   - Register cleanup via `tb.Cleanup(func() { ... })`
   - Use `tb.Errorf` (not Fatalf) inside the cleanup
   - The shim function is one line: forwards to the `*Context` variant
     with `tb.Context()`

5. **Add a smoke test** to `fixtures/fixtures_test.go` (or extend the
   existing one) that exercises seed+cleanup against a stub or LocalStack.
   Use `t.TempDir()` for any filesystem state. For real AWS interactions,
   gate the test behind `//go:build integration` and skip if Docker is
   unavailable.

6. **Run lint and test.**
   ```bash
   make lint
   make test-pkg PKG=./fixtures
   ```
   Common issues:
   - `thelper`: parameter named `t` instead of `tb`
   - `errcheck`: forgot to check return on cleanup call
   - `godot`: missing period at end of doc comment
   - `gosec G104`: explicit error checks needed (the cleanup template
     already handles this)

## Edge cases

- **Idempotent seed.** Some Put calls fail on existing-resource
  collisions. Either pre-delete (with cleanup pair anyway) or wrap with
  a "if not exists" guard. Document the behavior in the doc comment.
- **Cross-service fixtures.** A test may need to seed both an SSM param
  and an S3 object. Generate them as separate `Seed*` functions, not a
  single multi-service helper. Compose at the call site.
- **Force-delete.** Some services don't support immediate deletion
  (Secrets Manager has a 7-30 day recovery window). Use the force-delete
  variant in cleanup (e.g., `ForceDeleteWithoutRecovery: true` for
  Secrets Manager). Match the convention in `SeedSecret`.
- **Fixture without cleanup.** A few fixtures genuinely consume their
  payload (`SeedSQSMessage` — the test reads and removes it). For these,
  the doc comment must explicitly say "Cleanup: none (consumed by test)"
  and skip the `tb.Cleanup` registration. Rare; ask the user to confirm
  before going this route.

## References

- `references/fixture-template.go.tmpl` — full Seed function template
  with cleanup pair
- `fixtures/fixtures.go` — live file (read first to match conventions
  and ordering)
- `fixtures/fixtures_test.go` — existing tests (extend, don't duplicate)
