---
id: INV-0003
title: "Package documentation convention and gomarkdoc toolchain"
status: Concluded
author: Donald Gifford
created: 2026-05-13
---
<!-- markdownlint-disable-file MD025 MD041 -->

# INV 0003: Package documentation convention and gomarkdoc toolchain

**Status:** Concluded
**Author:** Donald Gifford
**Date:** 2026-05-13

<!--toc:start-->
- [Question](#question)
- [Hypothesis](#hypothesis)
- [Context](#context)
- [Approach](#approach)
- [Gap analysis: go-development plugin coverage](#gap-analysis-go-development-plugin-coverage)
- [Conclusion](#conclusion)
- [Resolved considerations](#resolved-considerations)
- [Recommendation](#recommendation)
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

## Gap analysis: go-development plugin coverage

The shared `go-development` plugin (loaded via the donald-loop
preamble; located at
`~/.claude/plugins/cache/donaldgifford-claude-skills/go-development/2.0.1/`)
references package documentation only obliquely:

| Coverage | What the plugin says | Verdict |
|----------|---------------------|---------|
| File-layout listing | `skills/go/references/project-structure.md:226` lists `doc.go` as one of the recommended package files: `doc.go            # Package-level documentation`. | **Mentioned, not specified.** |
| godoc content conventions | No reference file for `godoc` syntax, package-comment structure, or what content belongs in a package comment. | **Not covered.** |
| Required vs. optional doc.go | The file-layout listing reads as a suggestion, not a mandate. | **Not enforced.** |
| Rendering toolchain | No mention of `gomarkdoc`, `godoc -http`, or any markdown-renderer. | **Not covered.** |
| Enforcement | No lint hint or check for "package has a `doc.go`". | **Not covered.** |

**Conclusion of gap analysis.** The plugin acknowledges `doc.go`
exists but leaves four concrete gaps that this repo's convention
will fill (and may eventually push upstream into the plugin as
new reference files):

1. **A `doc.go` per-package mandate** — every Go package in the
   repo ships a `doc.go`, full stop.
2. **A content spec** — `doc.go` contains the `package <name>`
   declaration and a godoc-compliant multi-paragraph package
   comment. Imports, types, constants do not belong here.
3. **A renderer recommendation** — `gomarkdoc` (or equivalent)
   wired behind a `make docs` target. Out of scope for the initial
   convention; tracked separately.
4. **An enforcement check** — a small `scripts/check-doc-go.sh`
   (or Go program) that fails CI when a package directory lacks a
   `doc.go`. Out of scope for the initial convention; tracked
   separately.

## Conclusion

**Answer:** **Yes** — adopt the `doc.go`-per-package convention
immediately. Defer the rendering toolchain (`gomarkdoc`) and CI
enforcement to follow-up work; they're easier to land once the
convention is universal in the repo.

## Resolved considerations

- **doc.go for every package?** _Resolved._ **Yes — all packages,
  including `internal/`.** Consistency over special cases.
- **CI cost of gomarkdoc regen.** _Resolved._ **Manual `make docs`
  target only,** invoked before tagged releases (or by the release
  workflow after `Bump Version`). Not on every push — avoids noisy
  diffs and keeps `git status` clean during day-to-day development.
- **pkg.go.dev linking.** _Resolved._ Not a concern — out of scope
  for this convention. Consumers find us via pkg.go.dev naturally;
  the local renderer (when it lands) is for project docs.
- **Coupling to INV-0004.** _Resolved._ **Decouple.** The `doc.go`
  convention lands now as a settled rule; the rendering toolchain
  (`gomarkdoc`) and the Pro/OSS docgen tool (INV-0004) share
  docgen design space but ship in a separate DESIGN+IMPL cycle
  later.

## Recommendation

1. **Adopt the `doc.go` convention now** — fold the mechanical
   work (one `doc.go` per package, content lifted from the existing
   `// Package <name>` comments and expanded where intent is
   currently undocumented) into **IMPL-0004**. We're already
   touching every package in Phases 1–2 for the refactor; adding
   `doc.go` to the same pass is cheap.
2. **Document the convention in `CLAUDE.md`** under Code
   Conventions so every future Claude Code session enforces it
   without re-deriving from this INV.
3. **Defer tooling.** `gomarkdoc` wiring, the `make docs` target,
   and the CI doc.go-presence check are tracked separately —
   they're useful once the convention is universal, but they don't
   block IMPL-0004.
4. **Consider pushing the gap fixes upstream into the
   `go-development` plugin** as new reference files
   (`references/doc-go.md`, `references/package-comments.md`)
   after this convention has lived in the repo for a release or
   two.

## References

- [IMPL-0004 — Module hygiene primitives and per-service package
  layout][impl-0004] — Resolved Question 3 triggered this; the
  doc.go convention folds in here
- [DESIGN-0003 — Module hygiene primitives and per-service package
  layout][design-0003]
- [INV-0004 — Pro and OSS feature matrix tooling][inv-0004] —
  sibling investigation; shares docgen design space (decoupled)
- [`princjef/gomarkdoc`](https://github.com/princjef/gomarkdoc) —
  candidate renderer (deferred)
- [Effective Go — Commentary](https://go.dev/doc/effective_go#commentary)
  — godoc conventions
- `~/.claude/plugins/cache/donaldgifford-claude-skills/go-development/2.0.1/skills/go/references/project-structure.md`
  — current plugin coverage (line 226)

[impl-0004]: ../impl/0004-module-hygiene-primitives-and-per-service-package-layout.md
[design-0003]: ../design/0003-module-hygiene-primitives-and-per-service-package-layout.md
[inv-0004]: 0004-pro-and-oss-feature-matrix-tooling.md
