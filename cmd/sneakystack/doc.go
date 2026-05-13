// Package main is the entry point for the sneakystack standalone
// proxy binary.
//
// The binary ships as a multi-arch container image at
// ghcr.io/donaldgifford/sneakystack and exposes the same JSON-RPC /
// REST-XML surface as the embedded [sneakystack.Server] does inside
// libtftest's test harness. Use it when:
//
//   - A consumer test suite isn't built on libtftest but still
//     needs LocalStack-gap-filler endpoints
//   - A docker-compose stack wants sneakystack as a sidecar
//     alongside LocalStack and the Terraform-under-test
//   - CI wants a long-running sneakystack to test against, separate
//     from any individual test process
package main
