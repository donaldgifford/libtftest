package localstack

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteInitHooks(t *testing.T) {
	t.Parallel()

	hooks := []InitHook{
		{Name: "01-setup.sh", Script: "#!/bin/bash\necho setup"},
		{Name: "02-seed.sh", Script: "#!/bin/bash\necho seed"},
	}

	dir, err := WriteInitHooks(hooks)
	if err != nil {
		t.Fatalf("WriteInitHooks() error = %v", err)
	}
	defer os.RemoveAll(dir)

	for _, h := range hooks {
		path := filepath.Join(dir, h.Name)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read hook %s: %v", h.Name, err)
		}

		if string(data) != h.Script {
			t.Errorf("hook %s content = %q, want %q", h.Name, data, h.Script)
		}

		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat hook %s: %v", h.Name, err)
		}

		// Verify executable bit is set.
		if info.Mode()&0o100 == 0 {
			t.Errorf("hook %s mode = %o, want executable", h.Name, info.Mode())
		}
	}
}

func TestWriteInitHooks_Empty(t *testing.T) {
	t.Parallel()

	dir, err := WriteInitHooks(nil)
	if err != nil {
		t.Fatalf("WriteInitHooks(nil) error = %v", err)
	}
	defer os.RemoveAll(dir)

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}

	if len(entries) != 0 {
		t.Errorf("WriteInitHooks(nil) created %d files, want 0", len(entries))
	}
}
