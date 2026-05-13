package testfake_test

import (
	"testing"

	"github.com/donaldgifford/libtftest/internal/testfake"
)

func TestFakeTB_RecordsErrorf(t *testing.T) {
	t.Parallel()

	tb := testfake.NewFakeTB()
	tb.Errorf("ignored: %d", 1)

	if !tb.Errored() {
		t.Error("Errorf did not flip errored to true")
	}
	if tb.Fatalled() {
		t.Error("Errorf should not flip fatalled")
	}
	if tb.Skipped() {
		t.Error("Errorf should not flip skipped")
	}
}

func TestFakeTB_RecordsError(t *testing.T) {
	t.Parallel()

	tb := testfake.NewFakeTB()
	tb.Error("ignored")

	if !tb.Errored() {
		t.Error("Error did not flip errored to true")
	}
}

func TestFakeTB_RecordsFatalf(t *testing.T) {
	t.Parallel()

	tb := testfake.NewFakeTB()
	tb.Fatalf("ignored: %d", 1)

	if !tb.Fatalled() {
		t.Error("Fatalf did not flip fatalled to true")
	}
	if tb.Errored() {
		t.Error("Fatalf should not flip errored")
	}
}

func TestFakeTB_RecordsFatal(t *testing.T) {
	t.Parallel()

	tb := testfake.NewFakeTB()
	tb.Fatal("ignored")

	if !tb.Fatalled() {
		t.Error("Fatal did not flip fatalled to true")
	}
}

func TestFakeTB_RecordsSkip(t *testing.T) {
	t.Parallel()

	tb := testfake.NewFakeTB()
	tb.Skip("ignored")

	if !tb.Skipped() {
		t.Error("Skip did not flip skipped to true")
	}
}

func TestFakeTB_RecordsSkipf(t *testing.T) {
	t.Parallel()

	tb := testfake.NewFakeTB()
	tb.Skipf("ignored: %d", 1)

	if !tb.Skipped() {
		t.Error("Skipf did not flip skipped to true")
	}
}

func TestFakeTB_RecordsSkipNow(t *testing.T) {
	t.Parallel()

	tb := testfake.NewFakeTB()
	tb.SkipNow()

	if !tb.Skipped() {
		t.Error("SkipNow did not flip skipped to true")
	}
}

func TestFakeTB_RegistersCleanup(t *testing.T) {
	t.Parallel()

	tb := testfake.NewFakeTB()
	if tb.NumCleanups() != 0 {
		t.Fatalf("new FakeTB should have 0 cleanups, got %d", tb.NumCleanups())
	}

	tb.Cleanup(func() {})
	tb.Cleanup(func() {})

	if got, want := tb.NumCleanups(), 2; got != want {
		t.Errorf("NumCleanups = %d, want %d", got, want)
	}
}

func TestFakeTB_ZeroValueIsValid(t *testing.T) {
	t.Parallel()

	var tb testfake.FakeTB
	tb.Errorf("ignored")
	if !tb.Errored() {
		t.Error("zero-value FakeTB does not record Errorf")
	}
}

func TestFakeTB_ContextReturnsBackground(t *testing.T) {
	t.Parallel()

	tb := testfake.NewFakeTB()
	ctx := tb.Context()
	if ctx == nil {
		t.Fatal("Context returned nil")
	}
	if err := ctx.Err(); err != nil {
		t.Errorf("Context() should be live, got err: %v", err)
	}
}

func TestFakeTB_HelperLogLogfAreNoOps(t *testing.T) {
	t.Parallel()

	tb := testfake.NewFakeTB()
	tb.Helper()
	tb.Log("ignored")
	tb.Logf("ignored: %d", 1)

	if tb.Errored() || tb.Fatalled() || tb.Skipped() {
		t.Error("Helper/Log/Logf must not flip recording state")
	}
}
