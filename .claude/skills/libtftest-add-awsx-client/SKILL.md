---
name: libtftest:add-awsx-client
description: >
  Scaffold a new typed AWS SDK v2 client constructor in the awsx/ package
  of libtftest. Use when adding a New<Service>(cfg aws.Config) function,
  its imports, and a smoke test that match the existing convention.
when_to_use: >
  When adding support for a new AWS service to libtftest. Triggers on
  prompts like "add an awsx client for X", "we need an X client in awsx",
  or when a follow-up `libtftest:add-assertion` skill notices the client
  is missing.
allowed-tools: [Read, Write, Edit, Bash, Grep]
---

# libtftest:add-awsx-client

Adds a typed AWS SDK v2 client constructor to `awsx/clients.go`. Other
libtftest skills (`libtftest:add-assertion`, `libtftest:add-fixture`) depend
on these constructors, so build them first.

## Repo conventions (from `.claude/skills/_preamble.md`)

- Module path `github.com/donaldgifford/libtftest`
- Style: Uber Go Style Guide; line length 150; comments on exported symbols
  end with periods (`godot` linter)
- AWS SDK v2 v1.41.5: pass `aws.Config` **by value**. Use
  `<service>.NewFromConfig(cfg, …)`. Do **not** implement a custom
  `EndpointResolverV2` — `config.WithBaseEndpoint` (set in `awsx/config.go`)
  is what wires up the LocalStack edge URL.
- `gocritic.hugeParam` threshold raised to 800 in `.golangci.yml` so 696-byte
  `aws.Config` passes lint by-value.

## Procedure

When invoked, do the following in order:

1. **Confirm the service.** Ask the user for:
   - Service short name (e.g., `cloudwatch`, `events`)
   - AWS SDK v2 import path (e.g., `github.com/aws/aws-sdk-go-v2/service/cloudwatch`)
   - Constructor name (default: `New<TitleCase>`, e.g., `NewCloudWatch`)
   - Whether the service needs special options (e.g., S3 needs
     `UsePathStyle = true` for LocalStack — check the SDK docs if unsure)

2. **Read the current state of `awsx/clients.go`.** Place new constructors
   in alphabetical order by service name within the existing list.

3. **Add the import** to the import block in `awsx/clients.go`. Imports are
   alphabetical within the third-party group.

4. **Add the constructor** following the template in
   `references/awsx-client-template.go.tmpl`. The doc comment must end with
   a period.

5. **Run `go build ./awsx/`** to confirm it compiles. If the SDK module isn't
   downloaded yet, run `go mod tidy`.

6. **Run `make lint`** and fix any issues. Common issues:
   - Missing period at end of doc comment (`godot`)
   - Wrong import order (`gci`)

7. **Add a smoke test** to `awsx/clients_test.go` (create the file if it
   doesn't exist) that simply calls the constructor with an empty
   `aws.Config{}` and asserts the returned pointer is non-nil. Real
   integration is exercised by per-service tests, not here.

8. **Run `make test-pkg PKG=./awsx`** to confirm the smoke test passes.

## Edge cases

- **Service requires special client options.** Some services (S3, EC2)
  need extra `func(*<svc>.Options)` modifiers. If unsure, default to none
  and let the user adjust — note the addition in the PR description so a
  later reviewer can validate.
- **Service is Pro-only.** Many services (IAM enforcement, SSM Parameter
  Store, etc.) work fine on Community for client construction, but their
  *operations* may fail. The client constructor itself should not gate on
  edition. Edition gating belongs in `assert/` and tests, not here.
- **Service has multiple SDK packages.** Some services span multiple Go
  packages (e.g., `eventbridge` vs `events`). Use the canonical
  `service/<name>` path; ask the user if ambiguous.

## Reference

- `references/awsx-client-template.go.tmpl` — the constructor + doc comment
  template
- `awsx/clients.go` — the live file (read it before editing to match
  alphabetical ordering)
- `awsx/config.go` — where `config.WithBaseEndpoint` is wired (constructors
  do not need to repeat this)
