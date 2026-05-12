package fixtures

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

// fakeTB captures Fatalf/Errorf calls and registers cleanups against a
// channel so tests can drive them deterministically.
type fakeTB struct {
	testing.TB
	mu       sync.Mutex
	errored  bool
	fatalled bool
	cleanups []func()
}

func (*fakeTB) Helper() {}

func (f *fakeTB) Errorf(string, ...any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.errored = true
}

func (f *fakeTB) Error(...any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.errored = true
}

func (f *fakeTB) Fatalf(string, ...any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.fatalled = true
}

func (f *fakeTB) Fatal(...any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.fatalled = true
}

func (*fakeTB) Logf(string, ...any) {}

func (*fakeTB) Log(...any) {}

func (f *fakeTB) Cleanup(fn func()) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.cleanups = append(f.cleanups, fn)
}

func (*fakeTB) Context() context.Context {
	return context.Background()
}

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

// runWithTimeout runs fn in a goroutine and fails if it doesn't return.
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

func TestSeedS3ObjectContext_PropagatesCancel(t *testing.T) {
	t.Parallel()

	tb := &fakeTB{}
	runWithTimeout(t, func() {
		SeedS3ObjectContext(tb, cancelledCtx(t), testCfg(), "any-bucket", "any-key", []byte("body"))
	})

	if !tb.fatalled {
		t.Error("SeedS3ObjectContext with cancelled ctx did not Fatalf on the PutObject failure")
	}
}

func TestSeedSSMParameterContext_PropagatesCancel(t *testing.T) {
	t.Parallel()

	tb := &fakeTB{}
	runWithTimeout(t, func() {
		SeedSSMParameterContext(tb, cancelledCtx(t), testCfg(), "any-param", "value", false)
	})

	if !tb.fatalled {
		t.Error("SeedSSMParameterContext with cancelled ctx did not Fatalf")
	}
}

func TestSeedSecretContext_PropagatesCancel(t *testing.T) {
	t.Parallel()

	tb := &fakeTB{}
	runWithTimeout(t, func() {
		SeedSecretContext(tb, cancelledCtx(t), testCfg(), "any-secret", "value")
	})

	if !tb.fatalled {
		t.Error("SeedSecretContext with cancelled ctx did not Fatalf")
	}
}

func TestSeedSQSMessageContext_PropagatesCancel(t *testing.T) {
	t.Parallel()

	tb := &fakeTB{}
	runWithTimeout(t, func() {
		SeedSQSMessageContext(tb, cancelledCtx(t), testCfg(), "http://127.0.0.1:1/any-queue", "body")
	})

	if !tb.fatalled {
		t.Error("SeedSQSMessageContext with cancelled ctx did not Fatalf")
	}
}

// TestSeedS3ObjectContext_RegistersCleanup verifies that a cleanup
// callback is registered. The WithoutCancel behavior of that cleanup is
// guaranteed by the source — running it end-to-end requires LocalStack
// and is covered by the integration suite.
func TestSeedS3ObjectContext_RegistersCleanup(t *testing.T) {
	t.Parallel()

	tb := &fakeTB{}
	runWithTimeout(t, func() {
		SeedS3ObjectContext(tb, cancelledCtx(t), testCfg(), "any-bucket", "any-key", []byte("body"))
	})

	if len(tb.cleanups) == 0 {
		t.Error("SeedS3ObjectContext did not register a cleanup")
	}
}
