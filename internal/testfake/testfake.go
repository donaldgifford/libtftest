package testfake

import (
	"context"
	"sync"
	"testing"
)

// FakeTB is a minimal [testing.TB] stand-in that records whether the
// code under test reported a failure (Errorf/Error), aborted
// (Fatalf/Fatal), or skipped (Skip/Skipf/SkipNow), and which captures
// Cleanup registrations so callers can verify cleanup wiring without
// the real test runner.
//
// FakeTB embeds [testing.TB] so it satisfies the interface without
// implementing every method. Methods the embedded nil TB would
// satisfy at compile time but never get exercised at runtime in
// tests that use FakeTB.
//
// FakeTB is safe for concurrent use; assertions exercised under
// `go func() { ... }()` patterns may invoke any TB method from a
// goroutine.
type FakeTB struct {
	testing.TB

	mu       sync.Mutex
	errored  bool
	skipped  bool
	fatalled bool
	cleanups []func()
}

// NewFakeTB returns a FakeTB ready for use. The zero value of
// FakeTB is also valid; the constructor exists so test code reads
// `tb := testfake.NewFakeTB()` rather than `tb := &testfake.FakeTB{}`.
func NewFakeTB() *FakeTB {
	return &FakeTB{}
}

// Helper is a no-op matching [testing.TB.Helper].
func (*FakeTB) Helper() {}

// Errorf records that Errorf was called. The format and arguments
// are intentionally discarded — tests verify *that* a failure was
// reported, not what the message said.
func (f *FakeTB) Errorf(string, ...any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.errored = true
}

// Error records that Error was called.
func (f *FakeTB) Error(...any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.errored = true
}

// Fatalf records that Fatalf was called. Unlike the real
// [testing.TB.Fatalf], FakeTB does not call runtime.Goexit — the
// caller's goroutine continues so the test can observe the post-call
// state.
func (f *FakeTB) Fatalf(string, ...any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.fatalled = true
}

// Fatal records that Fatal was called.
func (f *FakeTB) Fatal(...any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.fatalled = true
}

// Skip records that Skip was called.
func (f *FakeTB) Skip(...any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.skipped = true
}

// Skipf records that Skipf was called.
func (f *FakeTB) Skipf(string, ...any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.skipped = true
}

// SkipNow records that SkipNow was called.
func (f *FakeTB) SkipNow() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.skipped = true
}

// Logf is a no-op matching [testing.TB.Logf]. Log output is
// uninteresting for the assertions exercised against FakeTB.
func (*FakeTB) Logf(string, ...any) {}

// Log is a no-op matching [testing.TB.Log].
func (*FakeTB) Log(...any) {}

// Cleanup records the supplied function so tests can verify that
// cleanup was registered. The function is NOT invoked — callers that
// want to exercise cleanup semantics should drive it from their own
// real [testing.T] via t.Cleanup or by calling fn directly.
func (f *FakeTB) Cleanup(fn func()) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.cleanups = append(f.cleanups, fn)
}

// Context returns a background context so FakeTB satisfies the
// tb.Context() callers in shim methods. Tests that exercise
// cancellation pass their own ctx directly to the *Context method,
// not through FakeTB.
func (*FakeTB) Context() context.Context {
	return context.Background()
}

// Errored reports whether Errorf or Error was called.
func (f *FakeTB) Errored() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.errored
}

// Skipped reports whether Skip, Skipf, or SkipNow was called.
func (f *FakeTB) Skipped() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.skipped
}

// Fatalled reports whether Fatalf or Fatal was called.
func (f *FakeTB) Fatalled() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.fatalled
}

// NumCleanups returns the number of Cleanup registrations.
func (f *FakeTB) NumCleanups() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.cleanups)
}
