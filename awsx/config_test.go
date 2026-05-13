package awsx

import (
	"context"
	"testing"
)

func TestNew_ReturnsConfig(t *testing.T) {
	t.Parallel()

	cfg, err := New(context.Background(), "http://localhost:4566")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if cfg.Region != "us-east-1" {
		t.Errorf("New().Region = %q, want us-east-1", cfg.Region)
	}

	creds, err := cfg.Credentials.Retrieve(context.Background())
	if err != nil {
		t.Fatalf("Credentials.Retrieve() error = %v", err)
	}

	if creds.AccessKeyID != "test" {
		t.Errorf("AccessKeyID = %q, want test", creds.AccessKeyID)
	}
}

func TestNewS3_PathStyle(t *testing.T) {
	t.Parallel()

	cfg, err := New(context.Background(), "http://localhost:4566")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	client := NewS3(cfg)
	if client == nil {
		t.Fatal("NewS3() returned nil")
	}
}

func TestNewDynamoDB(t *testing.T) {
	t.Parallel()

	cfg, err := New(context.Background(), "http://localhost:4566")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	client := NewDynamoDB(cfg)
	if client == nil {
		t.Fatal("NewDynamoDB() returned nil")
	}
}

func TestNewSTS(t *testing.T) {
	t.Parallel()

	cfg, err := New(context.Background(), "http://localhost:4566")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	client := NewSTS(cfg)
	if client == nil {
		t.Fatal("NewSTS() returned nil")
	}
}

func TestNewResourceGroupsTagging(t *testing.T) {
	t.Parallel()

	cfg, err := New(context.Background(), "http://localhost:4566")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	client := NewResourceGroupsTagging(cfg)
	if client == nil {
		t.Fatal("NewResourceGroupsTagging() returned nil")
	}
}
