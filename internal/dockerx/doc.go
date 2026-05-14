// Package dockerx provides Docker daemon detection and error
// classification helpers used by libtftest's container lifecycle
// code.
//
// The package answers two questions:
//
//   - "Is Docker reachable?" — Ping wraps the testcontainers-go
//     Reaper / Docker client probe with a libtftest-shaped error
//     so failures surface as actionable guidance ("Docker daemon
//     not reachable; start Docker Desktop") rather than raw socket
//     errors.
//
//   - "Is this a daemon-unavailable error or something else?" —
//     IsUnavailable classifies errors so the caller can decide
//     whether to t.Skip (CI without Docker) or t.Fatal (Docker
//     misconfiguration that the test author should fix).
//
// The package is internal because every consumer lives in
// libtftest's own packages — module authors should rely on the
// libtftest.New error path or the harness Run helpers instead.
package dockerx
