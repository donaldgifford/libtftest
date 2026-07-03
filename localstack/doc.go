// Package localstack manages LocalStack container lifecycle via
// testcontainers-go.
//
// The package wraps the testcontainers-go LocalStack module with
// libtftest-specific defaults:
//
//   - Image is pinned to localstack/localstack:2026.06.1 by default.
//     LocalStack ships a single, unified image using calendar
//     versioning (YYYY.MM.patch); Pro features are unlocked at
//     runtime via LOCALSTACK_AUTH_TOKEN rather than a separate
//     localstack-pro image. LIBTFTEST_LOCALSTACK_IMAGE or
//     Options.Image override the pin for custom or mirrored images
//   - Ports are bound via PortEndpoint with the explicit edge port
//     rather than Endpoint(http) — the latter picks the lowest port,
//     which is wrong for multi-port containers
//   - The AllServicesReady wait strategy uses the io.Reader signature
//     introduced in testcontainers-go v0.30
//   - LIBTFTEST_CONTAINER_URL bypasses container startup entirely
//     so a single external container can serve a whole test suite
//
// See INV-0002 for the Pro vs. OSS image-version landscape and
// DESIGN-0001 for the container-lifecycle modes the package
// supports (per-test, per-package via harness, per-suite via
// LIBTFTEST_CONTAINER_URL).
package localstack
