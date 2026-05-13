package secretsmanager_test

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"

	secretsfix "github.com/donaldgifford/libtftest/fixtures/secretsmanager"
	"github.com/donaldgifford/libtftest/internal/testfake"
)

func testCfg() aws.Config {
	return aws.Config{
		Region:       "us-east-1",
		BaseEndpoint: aws.String("http://127.0.0.1:1"),
		Credentials:  credentials.NewStaticCredentialsProvider("test", "test", ""),
	}
}

func cancelledCtx(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

func runWithTimeout(t *testing.T, fn func()) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		defer close(done)
		fn()
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("fixture did not return within 2s — ctx not honored")
	}
}

func TestSeedSecretContext_PropagatesCancel(t *testing.T) {
	t.Parallel()

	tb := testfake.NewFakeTB()
	runWithTimeout(t, func() {
		secretsfix.SeedSecretContext(tb, cancelledCtx(t), testCfg(), "any-secret", "value")
	})

	if !tb.Fatalled() {
		t.Error("SeedSecretContext with cancelled ctx did not Fatalf")
	}
}
