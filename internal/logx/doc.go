// Package logx provides structured logging configuration and test
// artifact dumping used by libtftest's TestCase lifecycle.
//
// The package centralises two concerns:
//
//   - slog Logger construction with the libtftest-specific
//     formatting (JSON for CI runs, human-readable for local), so
//     every package emits consistent output without re-deriving the
//     handler config.
//
//   - Test artifact dumping — when a test fails, libtftest dumps
//     LocalStack logs, container state, and workspace contents to
//     the test's t.TempDir for post-mortem inspection. logx exposes
//     the Dump helper that orchestrates this.
//
// The package is internal because the formatting choices and dump
// layout are not part of libtftest's public API surface.
package logx
