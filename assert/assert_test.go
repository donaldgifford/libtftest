package assert

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

// fakeTB captures Errorf/Fatalf/Skip calls so tests can verify the assertion
// reported a failure without exiting the real testing.T.
type fakeTB struct {
	testing.TB
	mu       sync.Mutex
	errored  bool
	skipped  bool
	fatalled bool
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

func (f *fakeTB) Skip(...any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.skipped = true
}

func (f *fakeTB) Skipf(string, ...any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.skipped = true
}

func (f *fakeTB) SkipNow() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.skipped = true
}

func (*fakeTB) Logf(string, ...any) {}

func (*fakeTB) Log(...any) {}

// Context returns a background context so the fake TB satisfies tb.Context()
// callers in shim methods. Tests that exercise cancellation pass their own
// ctx directly to the *Context method, not through the fake.
func (*fakeTB) Context() context.Context {
	return context.Background()
}

// testCfg returns an AWS config pointed at a non-existent endpoint so any
// network call fails quickly. Combined with a cancelled ctx, this is enough
// to verify the assertion records the error path.
func testCfg() aws.Config {
	return aws.Config{
		Region:       "us-east-1",
		BaseEndpoint: aws.String("http://127.0.0.1:1"),
		Credentials:  credentials.NewStaticCredentialsProvider("test", "test", ""),
	}
}

// cancelledCtx returns a context that is already cancelled.
func cancelledCtx(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

func TestS3_BucketExistsContext_PropagatesCancel(t *testing.T) {
	t.Parallel()

	tb := &fakeTB{}
	done := make(chan struct{})

	go func() {
		defer close(done)
		S3.BucketExistsContext(tb, cancelledCtx(t), testCfg(), "any-bucket")
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("BucketExistsContext did not honor cancelled ctx within 2s")
	}

	if !tb.errored {
		t.Error("BucketExistsContext with cancelled ctx did not report Errorf")
	}
}

func TestDynamoDB_TableExistsContext_PropagatesCancel(t *testing.T) {
	t.Parallel()

	tb := &fakeTB{}
	done := make(chan struct{})

	go func() {
		defer close(done)
		DynamoDB.TableExistsContext(tb, cancelledCtx(t), testCfg(), "any-table")
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("TableExistsContext did not honor cancelled ctx within 2s")
	}

	if !tb.errored {
		t.Error("TableExistsContext with cancelled ctx did not report Errorf")
	}
}

func TestSSM_ParameterExistsContext_PropagatesCancel(t *testing.T) {
	t.Parallel()

	tb := &fakeTB{}
	done := make(chan struct{})

	go func() {
		defer close(done)
		SSM.ParameterExistsContext(tb, cancelledCtx(t), testCfg(), "any-param")
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("ParameterExistsContext did not honor cancelled ctx within 2s")
	}

	if !tb.errored {
		t.Error("ParameterExistsContext with cancelled ctx did not report Errorf")
	}
}

func TestLambda_FunctionExistsContext_PropagatesCancel(t *testing.T) {
	t.Parallel()

	tb := &fakeTB{}
	done := make(chan struct{})

	go func() {
		defer close(done)
		Lambda.FunctionExistsContext(tb, cancelledCtx(t), testCfg(), "any-fn")
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("FunctionExistsContext did not honor cancelled ctx within 2s")
	}

	if !tb.errored {
		t.Error("FunctionExistsContext with cancelled ctx did not report Errorf")
	}
}

func TestIAM_RoleExistsContext_PropagatesCancel(t *testing.T) {
	// RoleExistsContext calls RequirePro internally, which skips when
	// LOCALSTACK_AUTH_TOKEN is unset. Set it so we exercise the ctx path.
	// t.Setenv requires the test not be parallel.
	t.Setenv("LOCALSTACK_AUTH_TOKEN", "test-token")

	tb := &fakeTB{}
	done := make(chan struct{})

	go func() {
		defer close(done)
		IAM.RoleExistsContext(tb, cancelledCtx(t), testCfg(), "any-role")
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("RoleExistsContext did not honor cancelled ctx within 2s")
	}

	if tb.skipped {
		t.Skip("RoleExistsContext was skipped by RequirePro — token env not honored?")
	}
	if !tb.errored {
		t.Error("RoleExistsContext with cancelled ctx did not report Errorf")
	}
}
