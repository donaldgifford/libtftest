---
id: INV-0004
title: "Pro and OSS feature matrix tooling"
status: Open
author: Donald Gifford
created: 2026-05-13
---
<!-- markdownlint-disable-file MD025 MD041 -->

# INV 0004: Pro and OSS feature matrix tooling

**Status:** Open
**Author:** Donald Gifford
**Date:** 2026-05-13

<!--toc:start-->
- [Question](#question)
- [Hypothesis](#hypothesis)
- [Context](#context)
- [Approach](#approach)
- [Candidate marker shapes](#candidate-marker-shapes)
  - [Option A: Structured comment](#option-a-structured-comment)
  - [Option B: Godoc-prefix convention](#option-b-godoc-prefix-convention)
  - [Option C: Centralised registry file](#option-c-centralised-registry-file)
- [Open considerations](#open-considerations)
- [References](#references)
<!--toc:end-->

## Question

What is the cleanest way to mark "this feature requires LocalStack
Pro" (or, conversely, "this feature is OSS-only") in the source so
that a small docgen tool can scan the codebase and render a
Pro / OSS feature matrix as markdown?

## Hypothesis

Build tags are the wrong tool here — they gate compilation, but we
don't want to *exclude* OSS users from compiling the Pro-only
helpers; we want consumers to *know* at design time which calls
will hit `t.Skip(...)` on OSS images via `libtftest.RequirePro`.

A lightweight structured comment marker (e.g.
`// libtftest:requires-pro <short reason>`) on functions/methods
that call `RequirePro(tb)` solves the design-time question without
changing build behaviour. A small docgen Go tool scans for the
marker, groups by package and service, and renders a markdown
matrix.

## Context

**Triggered by:** Donald's follow-up on IMPL-0004 — as more services
land (notably IAM Identity Center, Organizations, EKS pod identity
via `sneakystack`), consumers need a quick reference table to plan
which subset of `libtftest` they can use under OSS LocalStack vs.
Pro.

Today, the only way to discover Pro-gating is to read source for
`RequirePro(tb)` calls or hit the skip at runtime. Neither is
discoverable.

## Approach

1. Audit current `RequirePro` call sites — count how many helpers
   are Pro-gated today (IAM Identity Center handlers, Organizations
   handlers, EKS once we add it).
2. Pick a marker shape. Compare:
   - Custom structured comments (e.g.
     `// libtftest:requires-pro IAM IdC instance metadata`)
   - Custom Go struct tags on exported types (cannot apply to
     functions — non-starter)
   - Build tags (rejected per hypothesis; included for completeness)
   - A Go-doc convention (e.g. require `// Requires LocalStack Pro:`
     prefix on the godoc comment) — readable but harder to scan
     reliably
3. Build a minimum docgen tool: walks the `assert/`, `fixtures/`,
   `sneakystack/`, `libtftest.go` packages, scans for marker
   comments, emits a markdown table grouped by service.
4. Decide where the rendered matrix lives — likely
   `docs/pro-vs-oss.md`, linked from `README.md`.
5. Decide enforcement: should a missing marker on a function that
   calls `RequirePro` fail CI? Probably yes — that's the value of
   the marker convention.

## Candidate marker shapes

### Option A: Structured comment

```go
// PolicyAttachedToRole asserts ...
// libtftest:requires-pro IAM IdC managed policy ARNs only resolve under Pro
func PolicyAttachedToRole(tb testing.TB, ...) { ... }
```

- Pro: trivially greppable, survives `gofmt`, no AST work.
- Pro: same shape can encode reasons.
- Con: the linter has to understand a non-Go convention; needs a
  shared library file (similar to how we'd handle the
  `libtftest:` namespace).

### Option B: Godoc-prefix convention

```go
// Requires LocalStack Pro. PolicyAttachedToRole asserts ...
func PolicyAttachedToRole(tb testing.TB, ...) { ... }
```

- Pro: surfaces at pkg.go.dev for free.
- Con: hard to enforce reliably — godoc prefix variants drift
  ("Requires Pro", "LocalStack Pro only", etc.).

### Option C: Centralised registry file

`pro-features.go` (one per package) that lists exported symbols
gated by Pro.

- Pro: one place to look.
- Con: bit-rots when symbols are added without touching the
  registry; not co-located with the code.

Leaning toward **Option A** at draft time — the explicit marker is
greppable, machine-readable, and lets the docgen tool produce a
useful "Reason" column in the matrix.

## Open considerations

- **Coupling to INV-0003.** INV-0003 wants a docgen pipeline for
  package-level docs. Both investigations should converge on a
  single small Go tool that handles both, or use `gomarkdoc` for
  package docs + a separate tiny scanner for the Pro-OSS matrix.
- **CI enforcement.** Adding a `make check-pro-markers` target
  that fails if a function calls `RequirePro` without the marker
  comment ensures the matrix stays current. Open question whether
  this lives in `golangci-lint` (custom linter) or a standalone
  Go program.
- **Multi-tier gating.** Today we only have Pro vs. OSS. Future
  may add "Pro Enterprise" or "requires `LAMBDA_DOCKER_FLAGS` env
  var" or "requires `LOCALSTACK_AUTH_TOKEN`". Marker convention
  should leave room for additional categories without rework.

## References

- [IMPL-0004 — Module hygiene primitives and per-service package
  layout][impl-0004] — Future Work item 2 triggered this
- [DESIGN-0001 — libtftest shared terratest+LocalStack harness][design-0001]
  — defined the Pro/OSS edition split
- [INV-0003 — Package documentation convention and gomarkdoc
  toolchain][inv-0003] — sibling investigation; shares docgen
  design space
- `libtftest.RequirePro(tb)` — the current Pro-gate mechanism in
  the library

[impl-0004]: ../impl/0004-module-hygiene-primitives-and-per-service-package-layout.md
[design-0001]: ../design/0001-libtftest-shared-terratest-localstack-harness-for-aws-modules.md
[inv-0003]: 0003-package-documentation-convention-and-gomarkdoc-toolchain.md
