---
id: INV-0003
title: "Package documentation convention and gomarkdoc toolchain"
status: Open
author: Donald Gifford
created: 2026-05-13
---
<!-- markdownlint-disable-file MD025 MD041 -->

# INV 0003: Package documentation convention and gomarkdoc toolchain

**Status:** Open
**Author:** Donald Gifford
**Date:** 2026-05-13

<!--toc:start-->
- [Question](#question)
- [Hypothesis](#hypothesis)
- [Context](#context)
- [Approach](#approach)
- [Open considerations](#open-considerations)
- [References](#references)
<!--toc:end-->

## Question

Should this repo adopt a `doc.go`-per-package convention (with
godoc-compliant package comments) and a markdown-rendering toolchain
(`princjef/gomarkdoc` or similar) to surface the Go-doc API surface
as part of the docz docs under `docs/`?

## Hypothesis

Adopting both will give three returns from a single investment:

1. **Documented package intent.** Every package carries a written
   purpose alongside its types, removing the need to thread package
   intent through CHANGELOG entries or design docs (see IMPL-0004
   Resolved Question 3).
2. **Enforced godoc compliance.** The Uber Go style guide and the
   project's `godot` linter already nudge us toward godoc-compliant
   comments; a `doc.go` convention + a CI lint that requires
   `doc.go` to exist for every package raises this to a hard rule.
3. **A renderable, browsable API surface.** `gomarkdoc` renders Go
   doc comments to markdown; the output can live under `docs/api/`
   and be linked from the top-level `README.md`. Project newcomers
   read one rendered page instead of clicking package-by-package
   through pkg.go.dev.

## Context

**Triggered by:**

- IMPL-0004 Resolved Question 3 (the "deliberate non-change"
  CHANGELOG marker for `awsx/` — a `doc.go` package comment
  documents intent better than a synthetic chore commit)
- DESIGN-0003's per-service package split — once `assert/{service}/`
  and `fixtures/{service}/` exist, the number of packages roughly
  doubles. A per-package `doc.go` becomes more valuable.
- Prior art the project author has used: `princjef/gomarkdoc`
  (<https://github.com/princjef/gomarkdoc>) and custom docgen Go
  tools following the `docgen` idiom.

## Approach

1. Survey current packages — count those that already have a
   meaningful package-level comment (likely on the first `// Package
   <name>` line of an arbitrary `.go` file) vs those that don't.
2. Draft the `doc.go` convention: one file per package containing
   only the `package <name>` declaration and a multi-paragraph
   package comment.
3. Evaluate `gomarkdoc` for our needs: CLI ergonomics, output
   shape, ability to render multi-package navigation, integration
   with `make` or `mise`. Compare against custom-tool option.
4. Decide where the rendered markdown lives: `docs/api/` is the
   leading candidate; needs to coexist with the docz layout
   (RFC/ADR/Design/Impl/Investigation).
5. Decide on the lint enforcement story: pre-commit hook,
   `golangci-lint` custom linter, or a small `scripts/` Go program.

## Open considerations

- **Should every internal package require a `doc.go`?** Or only
  exported packages? Likely all packages — internals benefit
  equally from documented intent.
- **CI cost.** `gomarkdoc` regen on every push may cause noisy
  diffs. Consider running on tagged releases only, or behind a
  `make docs` target that users run manually before commit.
- **Linking to pkg.go.dev.** Even with local rendered markdown,
  consumers will still find us via pkg.go.dev. The local rendering
  is for project docs; pkg.go.dev is the canonical reference for
  external consumers.
- **Coupling to INV-0004.** The Pro/OSS feature matrix tool
  (INV-0004) wants a similar "scan the codebase, render markdown"
  shape. If we land a custom docgen for one, the other can
  piggyback.

## References

- [IMPL-0004 — Module hygiene primitives and per-service package
  layout][impl-0004] — Resolved Question 3 triggered this
- [DESIGN-0003 — Module hygiene primitives and per-service package
  layout][design-0003]
- [INV-0004 — Pro and OSS feature matrix tooling][inv-0004] —
  sibling investigation; shares docgen design space
- [`princjef/gomarkdoc`](https://github.com/princjef/gomarkdoc) —
  candidate toolchain
- [Effective Go — Commentary](https://go.dev/doc/effective_go#commentary)
  — godoc conventions

[impl-0004]: ../impl/0004-module-hygiene-primitives-and-per-service-package-layout.md
[design-0003]: ../design/0003-module-hygiene-primitives-and-per-service-package-layout.md
[inv-0004]: 0004-pro-and-oss-feature-matrix-tooling.md
