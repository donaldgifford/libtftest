package harness

import "testing"

func TestCurrent_NilWithoutRun(t *testing.T) {
	t.Parallel()

	got := Current()
	if got != nil {
		t.Errorf("Current() without Run = %v, want nil", got)
	}
}

func TestEdgeURL_EmptyWithoutRun(t *testing.T) {
	t.Parallel()

	got := EdgeURL()
	if got != "" {
		t.Errorf("EdgeURL() without Run = %q, want empty", got)
	}
}

func TestPrefixWarning_DetectsDuplicates(t *testing.T) {
	ResetPrefixes()
	defer ResetPrefixes()

	// First call should succeed.
	PrefixWarning(t, "ltt-abc123")

	// Check internal state for duplicate detection.
	prefixMu.Lock()
	hasDup := seenPrefixes["ltt-abc123"]
	prefixMu.Unlock()

	if !hasDup {
		t.Error("PrefixWarning() did not track prefix")
	}
}

func TestFormatContainerInfo_NoContainer(t *testing.T) {
	t.Parallel()

	got := FormatContainerInfo()
	if got != "no shared container" {
		t.Errorf("FormatContainerInfo() = %q, want 'no shared container'", got)
	}
}
