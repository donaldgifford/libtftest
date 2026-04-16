package tf

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildOptions_EnvVars(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	vars := map[string]any{"bucket_name": "test-bucket"}

	opts := BuildOptions(t, dir, vars)

	expectedEnvVars := map[string]string{
		"AWS_ACCESS_KEY_ID":     "test",
		"AWS_SECRET_ACCESS_KEY": "test",
		"AWS_DEFAULT_REGION":    "us-east-1",
		"TF_IN_AUTOMATION":      "1",
	}

	for key, want := range expectedEnvVars {
		if got, ok := opts.EnvVars[key]; !ok {
			t.Errorf("BuildOptions().EnvVars missing %q", key)
		} else if got != want {
			t.Errorf("BuildOptions().EnvVars[%q] = %q, want %q", key, got, want)
		}
	}

	// Plugin cache dir should be set.
	if _, ok := opts.EnvVars["TF_PLUGIN_CACHE_DIR"]; !ok {
		t.Error("BuildOptions().EnvVars missing TF_PLUGIN_CACHE_DIR")
	}
}

func TestBuildOptions_TerraformDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	opts := BuildOptions(t, dir, nil)

	if opts.TerraformDir != dir {
		t.Errorf("BuildOptions().TerraformDir = %q, want %q", opts.TerraformDir, dir)
	}
}

func TestBuildOptions_Vars(t *testing.T) {
	t.Parallel()

	vars := map[string]any{"bucket_name": "my-bucket"}
	opts := BuildOptions(t, t.TempDir(), vars)

	if opts.Vars["bucket_name"] != "my-bucket" {
		t.Errorf("BuildOptions().Vars[bucket_name] = %v, want my-bucket", opts.Vars["bucket_name"])
	}
}

func TestPluginCacheDir_Exists(t *testing.T) {
	t.Parallel()

	dir := PluginCacheDir()

	if dir == "" {
		t.Fatal("PluginCacheDir() returned empty string")
	}

	if !filepath.IsAbs(dir) {
		t.Errorf("PluginCacheDir() = %q, want absolute path", dir)
	}

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("PluginCacheDir() dir does not exist: %v", err)
	}

	if !info.IsDir() {
		t.Errorf("PluginCacheDir() = %q is not a directory", dir)
	}
}
