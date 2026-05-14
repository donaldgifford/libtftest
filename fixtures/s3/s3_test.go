package s3_test

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"

	s3fix "github.com/donaldgifford/libtftest/fixtures/s3"
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

func TestSeedObjectContext_PropagatesCancel(t *testing.T) {
	t.Parallel()

	tb := testfake.NewFakeTB()
	runWithTimeout(t, func() {
		s3fix.SeedObjectContext(tb, cancelledCtx(t), testCfg(), "any-bucket", "any-key", []byte("body"))
	})

	if !tb.Fatalled() {
		t.Error("SeedObjectContext with cancelled ctx did not Fatalf on the PutObject failure")
	}
}

// TestSeedObjectContext_RegistersCleanup verifies that a cleanup
// callback is registered. The WithoutCancel behavior of that cleanup
// is guaranteed by the source — running it end-to-end requires
// LocalStack and is covered by the integration suite.
func TestSeedObjectContext_RegistersCleanup(t *testing.T) {
	t.Parallel()

	tb := testfake.NewFakeTB()
	runWithTimeout(t, func() {
		s3fix.SeedObjectContext(tb, cancelledCtx(t), testCfg(), "any-bucket", "any-key", []byte("body"))
	})

	if tb.NumCleanups() == 0 {
		t.Error("SeedObjectContext did not register a cleanup")
	}
}
