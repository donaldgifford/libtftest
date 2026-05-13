package snapshot

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// updateEnv is the env var that, when set to "1", switches every
// JSON*-shaped helper into rewrite mode: missing or mismatched
// snapshot files are overwritten with the actual payload and the
// test passes (after a tb.Logf record).
const updateEnv = "LIBTFTEST_UPDATE_SNAPSHOTS"

// JSONStrict compares actual JSON bytes against the snapshot at path
// byte-for-byte. Failures call tb.Errorf with a short diff hint.
// When LIBTFTEST_UPDATE_SNAPSHOTS=1, a missing or mismatched snapshot
// is overwritten with actual and the test passes.
func JSONStrict(tb testing.TB, actual []byte, path string) {
	tb.Helper()
	compareSnapshot(tb, actual, path, false)
}

// JSONStructural normalizes actual and the snapshot (recursively sorts
// object keys and strips insignificant whitespace) before comparing.
// Use for IAM policies, plan JSON, and any payload whose key order is
// not semantically meaningful. When LIBTFTEST_UPDATE_SNAPSHOTS=1, a
// missing or mismatched snapshot is overwritten with the normalized
// actual and the test passes.
func JSONStructural(tb testing.TB, actual []byte, path string) {
	tb.Helper()
	compareSnapshot(tb, actual, path, true)
}

func compareSnapshot(tb testing.TB, actual []byte, path string, structural bool) {
	tb.Helper()

	update := os.Getenv(updateEnv) == "1"

	normalizedActual := actual
	if structural {
		n, err := NormalizeJSON(actual)
		if err != nil {
			tb.Errorf("snapshot: normalize actual: %v", err)
			return
		}
		normalizedActual = n
	}

	want, err := os.ReadFile(path)
	switch {
	case errors.Is(err, os.ErrNotExist):
		if !update {
			tb.Errorf("snapshot: %s missing; rerun with %s=1 to create it", path, updateEnv)
			return
		}
		writeSnapshot(tb, path, normalizedActual)
		tb.Logf("snapshot: wrote new %s (%d bytes)", path, len(normalizedActual))
		return
	case err != nil:
		tb.Errorf("snapshot: read %s: %v", path, err)
		return
	}

	gotForCompare := normalizedActual
	wantForCompare := want
	if structural {
		nw, err := NormalizeJSON(want)
		if err != nil {
			tb.Errorf("snapshot: normalize stored snapshot %s: %v", path, err)
			return
		}
		wantForCompare = nw
	}

	if bytes.Equal(gotForCompare, wantForCompare) {
		return
	}

	if update {
		writeSnapshot(tb, path, normalizedActual)
		tb.Logf("snapshot: rewrote %s (%d bytes)", path, len(normalizedActual))
		return
	}

	tb.Errorf("snapshot: %s mismatch\n--- want (%d bytes)\n+++ got (%d bytes)\n%s",
		path, len(wantForCompare), len(gotForCompare),
		shortDiffHint(wantForCompare, gotForCompare))
}

func writeSnapshot(tb testing.TB, path string, content []byte) {
	tb.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil { //nolint:gosec // Test snapshot directory.
		tb.Errorf("snapshot: mkdir %s: %v", filepath.Dir(path), err)
		return
	}
	if err := os.WriteFile(path, content, 0o600); err != nil {
		tb.Errorf("snapshot: write %s: %v", path, err)
	}
}

// NormalizeJSON parses raw and re-emits it with object keys sorted
// recursively and insignificant whitespace stripped. Exposed so
// callers can produce a normalized form for direct comparison without
// the snapshot-file plumbing.
func NormalizeJSON(raw []byte) ([]byte, error) {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	return marshalSorted(v)
}

func marshalSorted(v any) ([]byte, error) {
	switch x := v.(type) {
	case map[string]any:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		var buf bytes.Buffer
		buf.WriteByte('{')
		for i, k := range keys {
			if i > 0 {
				buf.WriteByte(',')
			}
			kb, err := json.Marshal(k)
			if err != nil {
				return nil, err
			}
			buf.Write(kb)
			buf.WriteByte(':')
			vb, err := marshalSorted(x[k])
			if err != nil {
				return nil, err
			}
			buf.Write(vb)
		}
		buf.WriteByte('}')
		return buf.Bytes(), nil

	case []any:
		var buf bytes.Buffer
		buf.WriteByte('[')
		for i, el := range x {
			if i > 0 {
				buf.WriteByte(',')
			}
			eb, err := marshalSorted(el)
			if err != nil {
				return nil, err
			}
			buf.Write(eb)
		}
		buf.WriteByte(']')
		return buf.Bytes(), nil

	default:
		return json.Marshal(v)
	}
}

// shortDiffHint returns a one-line summary of where want and got first
// diverge. The full diff lives in the snapshot file vs. the actual
// payload — surface enough to make the failure actionable without
// flooding test output.
func shortDiffHint(want, got []byte) string {
	limit := min(len(want), len(got))
	for i := range limit {
		if want[i] != got[i] {
			start := max(i-20, 0)
			endW := min(i+20, len(want))
			endG := min(i+20, len(got))
			return fmt.Sprintf("first diff at byte %d:\n  want: %q\n  got:  %q",
				i, want[start:endW], got[start:endG])
		}
	}
	if len(want) != len(got) {
		return fmt.Sprintf("length differs: want=%d got=%d", len(want), len(got))
	}
	return "<no byte-level diff but bytes.Equal returned false — corrupted input?>"
}
