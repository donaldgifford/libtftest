//go:build integration

package localstack

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"
)

func TestContainerStart_Community(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	cfg := &Config{
		Edition:  EditionCommunity,
		Image:    "localstack/localstack:4.4",
		Services: []string{"s3"},
	}

	ctr, err := Start(ctx, cfg)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer func() {
		if err := ctr.Stop(ctx); err != nil {
			t.Errorf("Stop() error = %v", err)
		}
	}()

	if ctr.ID == "" {
		t.Error("container ID is empty")
	}

	if ctr.EdgeURL == "" {
		t.Error("edge URL is empty")
	}

	if ctr.Edition != EditionCommunity {
		t.Errorf("edition = %v, want Community", ctr.Edition)
	}

	// Verify the container is actually healthy.
	resp, err := http.Get(ctr.EdgeURL + "/_localstack/health")
	if err != nil {
		t.Fatalf("health check request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("health check status = %d, want 200", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read health body: %v", err)
	}

	var hr HealthResponse
	if err := json.Unmarshal(body, &hr); err != nil {
		t.Fatalf("parse health response: %v", err)
	}

	t.Logf("LocalStack %s edition=%s services=%v", hr.Version, hr.Edition, hr.Services)
}

func TestContainerStart_ImageOverride(t *testing.T) {
	// This test verifies that LIBTFTEST_LOCALSTACK_IMAGE is respected.
	t.Setenv("LIBTFTEST_LOCALSTACK_IMAGE", "localstack/localstack:4.4")

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	cfg := &Config{
		Edition:  EditionCommunity,
		Services: []string{"s3"},
	}

	// Verify the image resolves to the env override.
	got := cfg.ResolveImage()
	if got != "localstack/localstack:4.4" {
		t.Errorf("ResolveImage() = %q, want localstack/localstack:4.4", got)
	}

	ctr, err := Start(ctx, cfg)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer func() {
		if err := ctr.Stop(ctx); err != nil {
			t.Errorf("Stop() error = %v", err)
		}
	}()

	if ctr.EdgeURL == "" {
		t.Error("edge URL is empty")
	}
}

func TestEditionDetection_FromHealthEndpoint(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	cfg := &Config{
		Edition:  EditionCommunity,
		Image:    "localstack/localstack:4.4",
		Services: []string{"s3"},
	}

	ctr, err := Start(ctx, cfg)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer func() {
		if err := ctr.Stop(ctx); err != nil {
			t.Errorf("Stop() error = %v", err)
		}
	}()

	resp, err := http.Get(ctr.EdgeURL + "/_localstack/health")
	if err != nil {
		t.Fatalf("health request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	edition := DetectEditionFromHealth(body)
	if edition != EditionCommunity {
		t.Errorf("DetectEditionFromHealth() = %v, want Community", edition)
	}
}
