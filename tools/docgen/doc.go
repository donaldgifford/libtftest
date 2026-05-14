// Command docgen scans the libtftest repo for
// `// libtftest:requires <tag>[,<tag>...] <reason>` marker comments,
// emits a JSON intermediate representation, renders the human-readable
// feature matrix to docs/feature-matrix.md, and gates CI by verifying
// that every function calling libtftest.RequirePro carries a marker.
//
// The tool is deliberately built so it has no compile-time dependency
// on any libtftest runtime package — it walks .go source files and uses
// only go/parser, go/ast, and the standard library. This keeps the
// tool's version coupling to libtftest source-level only: a layout
// change in libtftest can't break docgen, and a bump to docgen
// doesn't force a re-vendor of the rest of the module.
//
// # Subcommands
//
//	docgen scan   — write JSON IR to stdout (or -out path)
//	docgen render — read JSON IR (stdin or -in path); write Markdown to stdout (or -out path)
//	docgen check  — fail (exit 1) if any RequirePro caller lacks a marker
//
// # Marker grammar
//
//	// libtftest:requires <tag>[,<tag>...] <reason>
//
// Tags are comma-separated with no whitespace inside the list. The
// reason is free-form text and runs to end-of-line. Known tags today:
//
//   - pro    — function requires LocalStack Pro edition
//   - mockta — function depends on the external Okta mocking shim
//
// See docs/investigation/0004-pro-and-oss-feature-matrix-tooling.md
// for the marker grammar's design rationale.
package main
