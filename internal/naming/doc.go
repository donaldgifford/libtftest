// Package naming provides deterministic, parallel-safe resource
// name prefixes for libtftest's TestCase fixture.
//
// Every TestCase gets a unique 10-character prefix ("ltt-" plus six
// hex characters) that is:
//
//   - **Deterministic** — derived from a CSPRNG seed captured at
//     TestCase creation; the prefix is stable for the lifetime of
//     the TestCase.
//   - **Parallel-safe** — two parallel tests get distinct prefixes,
//     so AWS resource names (which share a flat namespace in
//     LocalStack) don't collide.
//   - **Short** — 10 chars stays inside the most aggressive AWS
//     resource-name length limits (e.g. S3 bucket names at 63
//     chars).
//
// The package is internal because the prefix scheme is part of
// libtftest's implementation contract, not its public API.
package naming
