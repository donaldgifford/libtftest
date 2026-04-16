package tf

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestNewWorkspace_CopiesFaithfully(t *testing.T) {
	t.Parallel()

	// Use testdata/mod-s3 as source.
	srcDir := filepath.Join("..", "testdata", "mod-s3")

	ws := NewWorkspace(t, srcDir)

	// Verify expected files exist in the copy.
	expectedFiles := []string{"main.tf", "variables.tf", "outputs.tf"}
	for _, f := range expectedFiles {
		path := filepath.Join(ws.Dir, f)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected file %s missing in workspace: %v", f, err)
		}
	}

	// Verify content matches.
	srcContent, err := os.ReadFile(filepath.Join(srcDir, "main.tf"))
	if err != nil {
		t.Fatalf("read source main.tf: %v", err)
	}

	dstContent, err := os.ReadFile(filepath.Join(ws.Dir, "main.tf"))
	if err != nil {
		t.Fatalf("read workspace main.tf: %v", err)
	}

	if !bytes.Equal(srcContent, dstContent) {
		t.Error("workspace main.tf content does not match source")
	}
}

func TestNewWorkspace_OriginalUntouched(t *testing.T) {
	t.Parallel()

	srcDir := filepath.Join("..", "testdata", "mod-s3")

	// Record original file count.
	srcEntries, err := os.ReadDir(srcDir)
	if err != nil {
		t.Fatalf("read source dir: %v", err)
	}
	originalCount := len(srcEntries)

	ws := NewWorkspace(t, srcDir)

	// Write a file in the workspace.
	testFile := filepath.Join(ws.Dir, "extra.tf")
	if err := os.WriteFile(testFile, []byte("# test"), 0o644); err != nil {
		t.Fatalf("write extra file: %v", err)
	}

	// Verify original dir is unchanged.
	afterEntries, err := os.ReadDir(srcDir)
	if err != nil {
		t.Fatalf("read source dir after: %v", err)
	}

	if len(afterEntries) != originalCount {
		t.Errorf("source dir file count changed: %d -> %d", originalCount, len(afterEntries))
	}
}

func TestCopyTree_NestedDirs(t *testing.T) {
	t.Parallel()

	src := t.TempDir()
	subDir := filepath.Join(src, "nested", "deep")
	if err := os.MkdirAll(subDir, 0o750); err != nil {
		t.Fatalf("create nested dir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(subDir, "file.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatalf("write nested file: %v", err)
	}

	dst := filepath.Join(t.TempDir(), "copy")

	if err := copyTree(src, dst); err != nil {
		t.Fatalf("copyTree() error = %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dst, "nested", "deep", "file.txt"))
	if err != nil {
		t.Fatalf("read nested copy: %v", err)
	}

	if string(got) != "hello" {
		t.Errorf("nested file content = %q, want %q", got, "hello")
	}
}
