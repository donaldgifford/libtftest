// Package harness provides shared-container TestMain helpers and
// the Sidecar interface for plugging auxiliary services (e.g.
// sneakystack) into the libtftest container lifecycle.
//
// # TestMain helpers
//
// The harness.Run function manages a shared LocalStack container
// across every test in a package, eliminating per-test container
// startup cost when tests are independent enough to share state.
// Callers wire it into TestMain:
//
//	func TestMain(m *testing.M) {
//		harness.Run(m, &libtftest.Options{...})
//	}
//
// # Sidecar interface
//
// Sidecar lets sneakystack — and any future LocalStack-gap-filler —
// participate in the same Start / Stop / Endpoint lifecycle that
// LocalStack uses, so the test fixture treats it as just another
// addressable AWS endpoint.
package harness
