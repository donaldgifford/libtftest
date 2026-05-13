// Package testfake provides a minimal in-memory [testing.TB] stand-in
// used by libtftest's per-service test packages to verify that
// assertions and fixtures route failures through the test handle
// rather than panicking, returning errors, or silently swallowing
// problems.
//
// FakeTB captures Errorf, Error, Fatalf, Fatal, Skip, Skipf, SkipNow,
// and Cleanup calls so a test can assert that an assertion under
// test reported the expected failure mode without exiting the real
// [testing.T] that drives the suite.
//
// The package is intentionally tiny and free of testing-helper
// dependencies. It is internal because every consumer lives in this
// module's own tests; libtftest consumers should use the standard
// library's [testing.T] directly.
//
// Typical use:
//
//	import "github.com/donaldgifford/libtftest/internal/testfake"
//
//	func TestFooContext_PropagatesCancel(t *testing.T) {
//		tb := testfake.NewFakeTB()
//		FooContext(tb, cancelledCtx(t), cfg, "name")
//		if !tb.Errored() {
//			t.Error("FooContext did not report Errorf on cancellation")
//		}
//	}
package testfake
