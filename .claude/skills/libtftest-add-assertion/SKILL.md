---
name: libtftest:add-assertion
description: >
  Scaffold a new per-service assertion package under assert/<service>/ in
  libtftest. Use when adding post-apply assertions for an AWS service that
  follow the per-service-package layout introduced in IMPL-0004
  (assert/s3, assert/dynamodb, assert/iam, ...).
when_to_use: >
  When the user says "add an assertion for X", "we need to assert that Y is
  Z after apply", or when implementing a new test pattern that doesn't have
  an assert/<service> helper yet. Activates inside the libtftest repo only.
allowed-tools: [Read, Write, Edit, Bash, Grep]
---

# libtftest:add-assertion

Adds (or extends) a per-service assertion package under `assert/<service>/`.
As of IMPL-0004 / DESIGN-0003 Part 1 the layout mirrors
`aws-sdk-go-v2/service/<name>`: each AWS service gets its own
`assert/<service>/` Go sub-package with **package-level functions**, not
methods on a zero-size struct. Function names drop the service prefix —
the package name carries it. Consumers import with an alias so the name
stays unambiguous alongside the AWS SDK:

```go
import s3assert "github.com/donaldgifford/libtftest/assert/s3"

s3assert.BucketExists(t, tc.AWS(), bucket)
s3assert.BucketExistsContext(t, ctx, tc.AWS(), bucket)
```

## Repo conventions

- `tb testing.TB` (not `t`) for any helper that takes a test handle —
  enforced by the `thelper` linter.
- Comments on exported symbols end with periods (`godot`).
- `aws.Config` is passed by value — `gocritic.hugeParam` threshold is 800.
- `aws.String(name)` for `*string` field assignments.
- Use `awsx.New<Service>(cfg)` for the AWS client. If the constructor
  doesn't exist yet, **first** run `/libtftest:add-awsx-client` and only
  then continue here.
- **Per-service-package layout (required as of v0.2.0).** One package
  per AWS service under `assert/<service>/`. The package name is the
  lowercase service short name (e.g., `s3`, `dynamodb`, `iam`). Function
  names drop the service prefix: it's `s3assert.BucketExists`, not
  `assert.S3.BucketExists`. The SDK package is imported under an alias
  (`s3sdk "github.com/aws/aws-sdk-go-v2/service/s3"`) to avoid collision
  with the package name.
- **Companion `doc.go` (per [INV-0003][inv-0003] / CLAUDE.md).** Every
  package ships a dedicated `doc.go` containing only the `package` line
  and a multi-paragraph godoc comment. No imports, types, or constants
  belong in `doc.go`.
- **Paired-method pattern (required as of v0.2.0).** Every assertion
  function has a `*Context` variant that accepts `context.Context`. The
  non-context function is a one-line shim that forwards with
  `tb.Context()`. Both forms are first-class.
- **Pro-only assertions** (anything requiring real AWS IAM enforcement,
  Organizations, SSO, etc.) call `libtftest.RequirePro(tb)` as the
  **first** line after `tb.Helper()`. They additionally carry a
  `// libtftest:requires pro <reason>` marker comment in the doc comment
  per [INV-0004][inv-0004] so `tools/docgen` can render the feature
  matrix. Multi-tag form (e.g. `pro,mockta`) is allowed.

## Procedure

1. **Confirm the service.** Ask the user for:
   - Service short name (e.g., `kms`, `cloudwatch`, `events`) — this is
     both the package name and the directory name
   - Function names + intent (e.g., `KeyExists(name)`,
     `KeyHasPolicy(name, policyJSON)`)
   - Edition: Community-only / Community + Pro / Pro-only. If unsure,
     consult `references/pro-services.md` for known Pro-gated services.
     Ask the user when ambiguous.

2. **Verify `awsx.New<Service>` exists.** Read `awsx/clients.go`. If the
   constructor is missing, stop and recommend:
   > "I need `awsx.New<Service>` first. Run `/libtftest:add-awsx-client`
   > with service=`<svc>`, then come back to this skill."

3. **Check whether `assert/<service>/` exists.**
   - If yes: add the new function(s) to the existing `<service>.go`
     file. Do not create a duplicate package.
   - If no: create the package directory with two files:
     - `assert/<service>/<service>.go` from
       `references/assertion-template.go.tmpl`
     - `assert/<service>/doc.go` — package line + godoc comment only

4. **Generate functions.** For each assertion:
   - Package-level function (no receiver).
   - First line: `tb.Helper()`. If Pro-only, the next line is
     `libtftest.RequirePro(tb)` AND the doc comment carries
     `// libtftest:requires pro <reason>`.
   - Generate both the `*Context` variant and the non-context shim. The
     shim is a one-liner that forwards to `*Context` with `tb.Context()`.
   - Use `tb.Errorf` (not `tb.Fatalf`) so multiple assertions can run.
   - Doc comment ends with a period and starts with the function name.
   - Import the AWS SDK service package under the `<service>sdk` alias
     to avoid collision with the package name itself.

5. **Generate test stubs.** Create `assert/<service>/<service>_test.go`
   in a `<service>_test` external test package. Use
   `internal/testfake.NewFakeTB()` for unit coverage — verify cancelled
   `ctx` propagation and error reporting. Real-LocalStack exercise
   belongs in `libtftest_integration_test.go` (or a service-specific
   integration test) behind `//go:build integration`.

6. **Run lint and test.**
   ```bash
   make lint
   make test-pkg PKG=./assert/<service>
   ```
   Common issues:
   - `godot`: missing period at end of doc comment
   - `thelper`: parameter named `t` instead of `tb`
   - `errcheck`: forgot to check the second return of an SDK call
   - `staticcheck`: unused imports if the function ended up not needing one
   - SDK package collides with the assert package name — use the `sdk`
     alias (e.g., `s3sdk`, `kmssdk`, `eventssdk`)

## Pro-vs-Community gating

When in doubt, ask the user. Indicators a service is Pro-only:

- Service name appears in `references/pro-services.md`
- The assertion exercises *enforcement* (e.g., "this IAM policy actually
  denies the action") — that's a Pro-only behavior. Pure existence checks
  on Pro-gated resources may also fail on Community.
- LocalStack docs mark the service "Pro" in their compatibility table.

If Pro-only, insert `libtftest.RequirePro(tb)` immediately after
`tb.Helper()` AND add the `// libtftest:requires pro <reason>` marker
to the doc comment:

```go
// KeyExistsContext asserts that the named KMS key exists.
//
// libtftest:requires pro LocalStack Pro for full KMS API coverage
func KeyExistsContext(tb testing.TB, ctx context.Context, cfg aws.Config, name string) {
    tb.Helper()
    libtftest.RequirePro(tb)
    // ...
}

// KeyExists is a shim that calls KeyExistsContext with tb.Context().
//
// libtftest:requires pro LocalStack Pro for full KMS API coverage
func KeyExists(tb testing.TB, cfg aws.Config, name string) {
    tb.Helper()
    KeyExistsContext(tb, tb.Context(), cfg, name)
}
```

The marker grammar is `// libtftest:requires <tag>[,<tag>...] <reason>`
— tags are comma-separated with no whitespace inside the list. Known
tags: `pro` (LocalStack Pro edition), `mockta` (Okta mocking shim).
`tools/docgen` consumes these to render `docs/feature-matrix.md`.

## References

- `references/assertion-template.go.tmpl` — full file template based on
  `assert/s3/s3.go`
- `references/pro-services.md` — known Pro-only AWS services for gating
  decisions
- `assert/s3/s3.go` — Community example (per-service package)
- `assert/iam/iam.go` — Pro-gated example (carries `libtftest:requires pro`)
- `internal/testfake/testfake.go` — shared fake `testing.TB` for unit tests

[inv-0003]: ../../../docs/investigation/0003-package-documentation-convention-and-gomarkdoc-toolchain.md
[inv-0004]: ../../../docs/investigation/0004-pro-and-oss-feature-matrix-tooling.md
