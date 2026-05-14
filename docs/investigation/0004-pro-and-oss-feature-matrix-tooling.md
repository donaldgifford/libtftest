---
id: INV-0004
title: "Pro and OSS feature matrix tooling"
status: Concluded
author: Donald Gifford
created: 2026-05-13
---
<!-- markdownlint-disable-file MD025 MD041 -->

# INV 0004: Pro and OSS feature matrix tooling

**Status:** Concluded
**Author:** Donald Gifford
**Date:** 2026-05-13

<!--toc:start-->
- [Question](#question)
- [Hypothesis](#hypothesis)
- [Context](#context)
  - [Known external dependencies that will need their own tag](#known-external-dependencies-that-will-need-their-own-tag)
  - [Multi-tag use case (concrete)](#multi-tag-use-case-concrete)
- [Approach](#approach)
- [Candidate marker shapes](#candidate-marker-shapes)
  - [Option A: Structured comment (chosen — multi-tag form)](#option-a-structured-comment-chosen--multi-tag-form)
  - [Option B: Godoc-prefix convention](#option-b-godoc-prefix-convention)
  - [Option C: Centralised registry file](#option-c-centralised-registry-file)
- [Resolved considerations](#resolved-considerations)
- [Recommendation](#recommendation)
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
will hit `t.Skip(...)` on OSS images via `libtftest.RequirePro` (or
analogous future gates like `RequireMockta`).

A lightweight structured comment marker that supports **a
comma-separated set of tags** rather than a single fixed tag:

```go
// libtftest:requires <tag>[,<tag>...] <short reason>
```

A small docgen Go tool scans for the marker, groups by package and
service, and renders a markdown matrix with one column per
encountered tag (today: `pro`, `mockta`; tomorrow: whatever else
we add).

## Context

**Triggered by:** Donald's follow-up on IMPL-0004 — as more services
land (notably IAM Identity Center, Organizations, EKS pod identity
via `sneakystack`), consumers need a quick reference table to plan
which subset of `libtftest` they can use under OSS LocalStack vs.
Pro.

Today, the only way to discover Pro-gating is to read source for
`RequirePro(tb)` calls or hit the skip at runtime. Neither is
discoverable.

### Known external dependencies that will need their own tag

The matrix is not strictly Pro-vs-OSS — it's "what external
auxiliary do I need to run this assertion?". Concrete examples:

| Tag | What it covers | Status |
|-----|----------------|--------|
| `pro` | Features that hit `libtftest.RequirePro(tb)` because they require a LocalStack Pro auth token | Today |
| `mockta` | Features that wrap **mockta**, an external Okta-mocking tool/service that `libtftest` will eventually wrap to test Okta-integrated modules | Planned |
| (future) | `auth-token`, `lambda-docker`, etc. as we add more gates | TBD |

The marker must accept arbitrary tag names so we can add new
external dependencies without changing the marker grammar.

### Multi-tag use case (concrete)

A single assertion may need both LocalStack Pro **and** mockta —
e.g. an EKS module that uses Okta for OIDC federation, where the
EKS side needs Pro and the Okta side needs mockta. That assertion
should be marked:

```go
// SomeOktaEKSAssertion asserts ...
// libtftest:requires pro,mockta Combines Pro EKS gates and mockta OIDC stubs
func SomeOktaEKSAssertion(tb testing.TB, ...) { ... }
```

The rendered matrix should show the assertion in **both** the
`pro` and `mockta` columns.

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

### Option A: Structured comment (chosen — multi-tag form)

```go
// PolicyAttachedToRole asserts ...
// libtftest:requires pro IAM IdC managed policy ARNs only resolve under Pro
func PolicyAttachedToRole(tb testing.TB, ...) { ... }
```

Multi-tag example:

```go
// OktaFederatedRoleHasTrust asserts ...
// libtftest:requires pro,mockta EKS + Okta federation needs both Pro and mockta
func OktaFederatedRoleHasTrust(tb testing.TB, ...) { ... }
```

Marker grammar:

```text
// libtftest:requires <tag>[,<tag>...] <free-text reason>
```

- Tags are a comma-separated list with no whitespace inside the
  list — keeps the regex simple and avoids ambiguity with the
  reason text.
- Reason is everything after the tag list. May contain commas, spaces,
  ASCII punctuation. Single line.
- Order within the tag list is not significant.
- Scanner emits one matrix row per function, one cell per
  distinct tag encountered across the codebase.

Why Option A:

- Pro: trivially greppable, survives `gofmt`, no AST work needed
  for a v1 scanner.
- Pro: same shape encodes reasons and arbitrary tags.
- Pro: extensible without changing the marker grammar — adding a
  new external dependency tag (e.g. `lambda-docker`) is just a
  marker-text change.
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

## Resolved considerations

- **Marker shape.** _Resolved._ Option A — structured comment with
  comma-separated multi-tag grammar
  (`// libtftest:requires <tag>[,<tag>...] <reason>`). Greppable,
  no AST work needed for the v1 scanner, extensible without
  grammar changes.
- **Coupling to INV-0003.** _Resolved._ The marker scanner is
  intentionally decoupled from any AST/import-time work — it's a
  regex pass over Go source files. That means it does NOT need to
  share infrastructure with the gomarkdoc-style package-doc
  renderer (INV-0003's deferred tool). They may eventually live
  under `tools/` together, but they're independent programs.
- **Version sync.** _Resolved._ The scanner is version-agnostic by
  design — it doesn't import any libtftest packages, just scans
  comment text. So it can ship in this repo (under `tools/docgen/`)
  without needing to track library version per build.
- **CI enforcement.** _Resolved._ Standalone Go program (no
  `golangci-lint` custom linter), invoked via
  `make check-markers`. Fails when a function calls
  `libtftest.RequirePro(tb)` (detected by a regex over the same
  source files) without an accompanying
  `// libtftest:requires ...` marker on the same function. Wired
  into `make ci` so PR CI catches missing markers.
- **Multi-tier / multi-dependency gating.** _Resolved._ The
  marker accepts a comma-separated list of arbitrary tags, so
  adding a new tier or external dependency (mockta, lambda-docker,
  Pro Enterprise, etc.) is just a marker-text change with no
  grammar rework.
- **Output location.** _Resolved._ Rendered matrix lives at
  `docs/feature-matrix.md`, linked from `README.md`. Regenerated
  via `make docs-matrix`. Not regenerated on every push — only
  before tagged releases (and committed as `docs(feature-matrix):
  regenerate for v<x.y.z>`).

## Recommendation

Both the marker convention and the scanner/CI gate fold into
IMPL-0004 as a new Phase 9. The rendered matrix file
(`docs/feature-matrix.md`) ships with the v0.2.0 release so
consumers see immediate value alongside the layout refactor and
new primitives.

Pushing the marker grammar upstream into the `go-development`
plugin (or as a shared convention across the donaldgifford
toolbox) is deferred — let it bake in this repo for a release
before generalising.

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
