package logx

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestNewLogger(t *testing.T) {
	t.Parallel()

	logger := NewLogger(t)
	if logger == nil {
		t.Fatal("NewLogger(t) returned nil")
	}
}

func TestDumpArtifact_WritesToDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	data := []byte("test artifact content")

	DumpArtifact(t, dir, "test.txt", data)

	got, err := os.ReadFile(filepath.Join(dir, "test.txt"))
	if err != nil {
		t.Fatalf("read artifact: %v", err)
	}

	if !bytes.Equal(got, data) {
		t.Errorf("artifact content = %q, want %q", got, data)
	}
}

func TestDumpArtifact_WritesToCIDir(t *testing.T) {
	ciDir := t.TempDir()
	t.Setenv("LIBTFTEST_ARTIFACT_DIR", ciDir)

	localDir := t.TempDir()
	data := []byte("ci artifact content")

	DumpArtifact(t, localDir, "ci-test.txt", data)

	// Verify written to both local and CI dirs.
	localPath := filepath.Join(localDir, "ci-test.txt")
	if _, err := os.Stat(localPath); err != nil {
		t.Errorf("local artifact missing: %v", err)
	}

	ciPath := filepath.Join(ciDir, t.Name(), "ci-test.txt")
	got, err := os.ReadFile(ciPath)
	if err != nil {
		t.Fatalf("read CI artifact: %v", err)
	}

	if !bytes.Equal(got, data) {
		t.Errorf("CI artifact content = %q, want %q", got, data)
	}
}

func TestResolveArtifactDir_DefaultsToBaseDir(t *testing.T) {
	t.Setenv("LIBTFTEST_ARTIFACT_DIR", "")

	baseDir := filepath.Join(os.TempDir(), "test-base")
	dir := ResolveArtifactDir(t, baseDir)
	want := filepath.Join(baseDir, "libtftest-artifacts")

	if dir != want {
		t.Errorf("ResolveArtifactDir() = %q, want %q", dir, want)
	}
}

func TestResolveArtifactDir_RespectsEnvVar(t *testing.T) {
	artifactDir := filepath.Join(os.TempDir(), "my-artifacts")
	t.Setenv("LIBTFTEST_ARTIFACT_DIR", artifactDir)

	dir := ResolveArtifactDir(t, filepath.Join(os.TempDir(), "ignored"))
	want := filepath.Join(artifactDir, t.Name())

	if dir != want {
		t.Errorf("ResolveArtifactDir() = %q, want %q", dir, want)
	}
}
