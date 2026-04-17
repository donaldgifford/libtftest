package tf

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRenderProviderOverride_ValidJSON(t *testing.T) {
	t.Parallel()

	data, err := RenderProviderOverride("http://localhost:4566")
	if err != nil {
		t.Fatalf("RenderProviderOverride() error = %v", err)
	}

	// Verify it's valid JSON.
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	// Verify structure.
	provider, ok := parsed["provider"].(map[string]any)
	if !ok {
		t.Fatal("missing provider key")
	}

	aws, ok := provider["aws"].(map[string]any)
	if !ok {
		t.Fatal("missing aws key")
	}

	endpoints, ok := aws["endpoints"].(map[string]any)
	if !ok {
		t.Fatal("missing endpoints key")
	}

	// Verify all services point to the edge URL.
	for _, svc := range services {
		url, ok := endpoints[svc].(string)
		if !ok {
			t.Errorf("service %q missing from endpoints", svc)
			continue
		}

		if url != "http://localhost:4566" {
			t.Errorf("endpoints[%q] = %q, want http://localhost:4566", svc, url)
		}
	}
}

func TestRenderProviderOverride_DynamicPort(t *testing.T) {
	t.Parallel()

	data, err := RenderProviderOverride("http://localhost:49312")
	if err != nil {
		t.Fatalf("RenderProviderOverride() error = %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	provider := parsed["provider"].(map[string]any)
	aws := provider["aws"].(map[string]any)
	endpoints := aws["endpoints"].(map[string]any)

	// Verify the dynamic port is used.
	if endpoints["s3"] != "http://localhost:49312" {
		t.Errorf("endpoints[s3] = %v, want http://localhost:49312", endpoints["s3"])
	}
}

func TestRenderBackendOverride_ValidJSON(t *testing.T) {
	t.Parallel()

	data := RenderBackendOverride()

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	terraform, ok := parsed["terraform"].(map[string]any)
	if !ok {
		t.Fatal("missing terraform key")
	}

	backend, ok := terraform["backend"].(map[string]any)
	if !ok {
		t.Fatal("missing backend key")
	}

	if _, ok := backend["local"]; !ok {
		t.Error("missing local backend key")
	}
}

func TestWriteOverrides(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	if err := WriteOverrides(dir, "http://localhost:4566"); err != nil {
		t.Fatalf("WriteOverrides() error = %v", err)
	}

	// Verify provider override file exists and is valid JSON.
	providerData, err := os.ReadFile(filepath.Join(dir, providerOverrideFile))
	if err != nil {
		t.Fatalf("read provider override: %v", err)
	}

	var provider map[string]any
	if err := json.Unmarshal(providerData, &provider); err != nil {
		t.Errorf("provider override invalid JSON: %v", err)
	}

	// Verify backend override file exists and is valid JSON.
	backendData, err := os.ReadFile(filepath.Join(dir, backendOverrideFile))
	if err != nil {
		t.Fatalf("read backend override: %v", err)
	}

	var backend map[string]any
	if err := json.Unmarshal(backendData, &backend); err != nil {
		t.Errorf("backend override invalid JSON: %v", err)
	}
}
