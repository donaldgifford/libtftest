// Package main is the entry point for the libtftest CLI.
//
// The CLI is a thin wrapper around the library — its primary use
// case is one-off introspection of a libtftest-aware test suite:
// container endpoint discovery, manifest dumping, and version
// reporting. The library API is what test code uses; the CLI is
// for operators and integrations.
package main
