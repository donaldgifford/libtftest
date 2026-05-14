// Package sneakystack provides a LocalStack gap-filling HTTP proxy
// with an in-memory store, designed to surface AWS APIs that
// LocalStack itself does not cover (IAM Identity Center,
// Organizations, Control Tower, etc.).
//
// The package exposes:
//
//   - A Server that fronts both JSON-RPC (AWS json-1.1) and
//     REST-XML protocols, dispatching to per-service handlers
//     registered under sneakystack/services
//
//   - A Store interface backed by plain Go maps (no external DB
//     dependency); handlers stash and retrieve their state via the
//     Store, keeping the package import-cycle-clean
//
//   - A Sidecar implementation that lets libtftest's harness drive
//     sneakystack alongside LocalStack as a single addressable
//     endpoint
//
// sneakystack also ships as a standalone Docker container
// (cmd/sneakystack), so consumer test suites that aren't using
// libtftest can still benefit from the gap-fillers.
//
// See DESIGN-0001 for the Sidecar architecture and the rationale
// for plain Go maps over go-memdb.
package sneakystack
