package snapshot_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/donaldgifford/libtftest/assert/snapshot"
	"github.com/donaldgifford/libtftest/internal/testfake"
)

func writeFile(t *testing.T, path string, content []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestJSONStrict_Identical(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "s.json")
	doc := []byte(`{"a":1,"b":2}`)
	writeFile(t, path, doc)

	tb := testfake.NewFakeTB()
	snapshot.JSONStrict(tb, doc, path)

	if tb.Errored() {
		t.Errorf("strict on identical payloads should pass; got Errorf")
	}
}

func TestJSONStrict_BytewiseDifferentSameStructure(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "s.json")
	writeFile(t, path, []byte(`{"a":1,"b":2}`))

	tb := testfake.NewFakeTB()
	snapshot.JSONStrict(tb, []byte(`{"b":2,"a":1}`), path)

	if !tb.Errored() {
		t.Error("strict on reordered keys should Errorf")
	}
}

func TestJSONStructural_BytewiseDifferentSameStructure(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "s.json")
	writeFile(t, path, []byte(`{"a":1,"b":2}`))

	tb := testfake.NewFakeTB()
	snapshot.JSONStructural(tb, []byte(`{"b":2,"a":1}`), path)

	if tb.Errored() {
		t.Error("structural on reordered keys should pass; got Errorf")
	}
}

func TestJSONStructural_DifferentValues(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "s.json")
	writeFile(t, path, []byte(`{"a":1,"b":2}`))

	tb := testfake.NewFakeTB()
	snapshot.JSONStructural(tb, []byte(`{"a":1,"b":3}`), path)

	if !tb.Errored() {
		t.Error("structural on different values should Errorf")
	}
}

func TestJSONStructural_MissingSnapshot_NoUpdate(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "missing.json")

	tb := testfake.NewFakeTB()
	snapshot.JSONStructural(tb, []byte(`{"a":1}`), path)

	if !tb.Errored() {
		t.Error("missing snapshot without update mode should Errorf")
	}
	if _, err := os.Stat(path); err == nil {
		t.Error("missing snapshot without update mode should not write the file")
	}
}

func TestJSONStructural_MissingSnapshot_WithUpdate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "missing.json")

	t.Setenv("LIBTFTEST_UPDATE_SNAPSHOTS", "1")
	tb := testfake.NewFakeTB()
	snapshot.JSONStructural(tb, []byte(`{"b":2,"a":1}`), path)

	if tb.Errored() {
		t.Errorf("missing snapshot in update mode should pass; got Errorf")
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("snapshot file not written: %v", err)
	}
	// Update mode writes the *normalized* payload — keys must be sorted.
	const want = `{"a":1,"b":2}`
	if string(got) != want {
		t.Errorf("written snapshot = %q, want %q (normalized)", string(got), want)
	}
}

func TestJSONStrict_MissingSnapshot_WithUpdate_WritesRaw(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "missing.json")

	t.Setenv("LIBTFTEST_UPDATE_SNAPSHOTS", "1")
	tb := testfake.NewFakeTB()
	const payload = `{"b":2,"a":1}`
	snapshot.JSONStrict(tb, []byte(payload), path)

	if tb.Errored() {
		t.Errorf("strict update should pass; got Errorf")
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("snapshot file not written: %v", err)
	}
	if string(got) != payload {
		t.Errorf("strict update writes raw payload as-is; got %q, want %q",
			string(got), payload)
	}
}

func TestJSONStructural_MismatchedWithUpdate_RewritesPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "s.json")
	writeFile(t, path, []byte(`{"a":1}`))

	t.Setenv("LIBTFTEST_UPDATE_SNAPSHOTS", "1")
	tb := testfake.NewFakeTB()
	snapshot.JSONStructural(tb, []byte(`{"a":1,"b":2}`), path)

	if tb.Errored() {
		t.Errorf("mismatch in update mode should pass; got Errorf")
	}
	got, _ := os.ReadFile(path)
	if string(got) != `{"a":1,"b":2}` {
		t.Errorf("snapshot was not rewritten; got %q", string(got))
	}
}

func TestJSONStrict_InvalidActualJSON(t *testing.T) {
	// JSONStrict doesn't parse — it byte-compares — so this is just a
	// snapshot-missing scenario from the strict path's perspective.
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "s.json")
	writeFile(t, path, []byte("not-json"))

	tb := testfake.NewFakeTB()
	snapshot.JSONStrict(tb, []byte("not-json"), path)

	if tb.Errored() {
		t.Error("strict tolerates non-JSON if bytes match")
	}
}

func TestNormalizeJSON_SortsKeys(t *testing.T) {
	t.Parallel()

	out, err := snapshot.NormalizeJSON([]byte(`{"b":1,"a":{"d":2,"c":3}}`))
	if err != nil {
		t.Fatal(err)
	}
	const want = `{"a":{"c":3,"d":2},"b":1}`
	if string(out) != want {
		t.Errorf("NormalizeJSON = %q, want %q", string(out), want)
	}
}

func TestNormalizeJSON_InvalidJSON(t *testing.T) {
	t.Parallel()

	_, err := snapshot.NormalizeJSON([]byte("not-json"))
	if err == nil {
		t.Error("NormalizeJSON on invalid input should error")
	}
}
