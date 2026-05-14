// Package tools is a parent directory marker for repo-local Go
// tooling that is built but not redistributed as part of the public
// libtftest module surface.
//
// Each sub-directory is its own `package main` Go program. Tools live
// here so they can `import` libtftest packages cheaply during
// development without leaking into the redistributable surface — the
// `tools/...` import path is reserved for build-time helpers, never
// runtime dependencies.
//
// Current tools:
//
//   - tools/docgen — scans the repo for // libtftest:requires markers,
//     renders the feature matrix in docs/feature-matrix.md, and gates
//     CI on every RequirePro caller having a marker. See
//     tools/docgen/doc.go for details.
package tools
