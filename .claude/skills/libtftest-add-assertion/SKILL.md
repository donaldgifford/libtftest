---
name: libtftest:add-assertion
description: >
  Scaffold a new assertion namespace and methods in the assert/ package of
  libtftest. Use when adding post-apply assertions for an AWS service that
  follow the zero-size struct + package-level var pattern (assert/s3.go).
when_to_use: >
  When the user says "add an assertion for X", "we need to assert that Y is
  Z after apply", or when implementing a new test pattern that doesn't have
  an assert.* helper yet. Activates inside the libtftest repo only.
allowed-tools: [Read, Write, Edit, Bash, Grep]
---

# libtftest:add-assertion

Adds a new assertion namespace under `assert/`. The pattern is a zero-size
unexported struct type (e.g., `kmsAsserts`) with methods, plus a package-level
`var <Service> <service>Asserts` declared in `assert/assert.go`. Tests use it
as `assert.<Service>.<Method>(t, tc.AWS(), …)`.

## Repo conventions

- `tb testing.TB` (not `t`) for any helper that takes a test handle —
  enforced by the `thelper` linter.
- Comments on exported symbols end with periods (`godot`).
- `aws.Config` is passed by value — `gocritic.hugeParam` threshold is 800.
- `aws.String(name)` for `*string` field assignments.
- Use `awsx.New<Service>(cfg)` for the AWS client. If the constructor
  doesn't exist yet, **first** run `/libtftest:add-awsx-client` and only
  then continue here.
- Pro-only assertions (anything that requires real AWS IAM enforcement,
  Organizations, SSO, etc.) call `libtftest.RequirePro(tb)` as the **first**
  line after `tb.Helper()`. This skips the test on Community edition with
  a clear message instead of failing.
- File organization: one `assert/<service>.go` file per service, all
  methods live on a single struct receiver.
- **Paired-method pattern (required as of v0.2.0).** Every assertion
  method has a `*Context` variant that accepts `context.Context`. The
  non-context method is a one-line shim that forwards with
  `tb.Context()`. Both forms are first-class. Generated tests should
  exercise the `*Context` variant since it's the canonical form;
  the shim is structurally trivial.

## Procedure

1. **Confirm the service.** Ask the user for:
   - Service short name (e.g., `kms`, `cloudwatch`, `events`)
   - Method names + intent (e.g., `KeyExists(name)`, `KeyHasPolicy(name, policyJSON)`)
   - Edition: Community-only / Community + Pro / Pro-only. If unsure,
     consult `references/pro-services.md` for known Pro-gated services.
     Ask the user when ambiguous.

2. **Verify `awsx.New<Service>` exists.** Read `awsx/clients.go`. If the
   constructor is missing, stop and recommend:
   > "I need `awsx.New<Service>` first. Run `/libtftest:add-awsx-client`
   > with service=`<svc>`, then come back to this skill."

3. **Check whether `assert/<service>.go` exists.**
   - If yes: add the new method(s) to the existing struct. Do not create
     a duplicate file.
   - If no: create `assert/<service>.go` from
     `references/assertion-template.go.tmpl` and add a new package-level
     var to `assert/assert.go` in alphabetical order.

4. **Generate methods.** For each method:
   - Receiver is `(<service>Asserts)` (no name — it's a zero-size type).
   - First line: `tb.Helper()`. If Pro-only, the next line is
     `libtftest.RequirePro(tb)`.
   - Use `tb.Errorf` (not `tb.Fatalf`) so multiple assertions can run.
   - Doc comment ends with a period and starts with the method name.

5. **Generate test stubs.** Create `assert/<service>_test.go` (or extend it)
   with a table-driven test for each method. Stubs use `t.Run(name, func)`
   subtests; each calls the assertion and verifies expected behavior on a
   fake/mocked client OR documents that the real exercise lives in the
   integration test under `assert/<service>_integration_test.go`.

6. **Run lint and test.**
   ```bash
   make lint
   make test-pkg PKG=./assert
   ```
   Common issues:
   - `godot`: missing period at end of doc comment
   - `thelper`: parameter named `t` instead of `tb`
   - `errcheck`: forgot to check the second return of an SDK call
   - `staticcheck`: unused imports if the method ended up not needing one

## Pro-vs-Community gating

When in doubt, ask the user. Indicators a service is Pro-only:

- Service name appears in `references/pro-services.md`
- The assertion exercises *enforcement* (e.g., "this IAM policy actually
  denies the action") — that's a Pro-only behavior. Pure existence checks
  on Pro-gated resources may also fail on Community.
- LocalStack docs mark the service "Pro" in their compatibility table.

If Pro-only, insert `libtftest.RequirePro(tb)` immediately after
`tb.Helper()`. Document the Pro requirement in the method doc comment:

```go
// KeyExistsContext is the ctx-aware variant of KeyExists. Pro-only: calls RequirePro.
func (kmsAsserts) KeyExistsContext(tb testing.TB, ctx context.Context, cfg aws.Config, name string) {
    tb.Helper()
    libtftest.RequirePro(tb)
    // …
}

// KeyExists is a shim that calls KeyExistsContext with tb.Context().
func (k kmsAsserts) KeyExists(tb testing.TB, cfg aws.Config, name string) {
    tb.Helper()
    k.KeyExistsContext(tb, tb.Context(), cfg, name)
}
```

## References

- `references/assertion-template.go.tmpl` — full file template based on
  `assert/s3.go`
- `references/pro-services.md` — known Pro-only AWS services for gating
  decisions
- `assert/s3.go` — Community example
- `assert/iam.go` — Pro-gated example
- `assert/assert.go` — where to add the package-level `var <Service>`
