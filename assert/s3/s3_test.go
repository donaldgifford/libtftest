package s3_test

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"

	s3assert "github.com/donaldgifford/libtftest/assert/s3"
	"github.com/donaldgifford/libtftest/internal/testfake"
)

// testCfg returns an AWS config pointed at a non-existent endpoint so
// any network call fails quickly. Combined with a cancelled ctx, this
// is enough to verify the assertion records the error path.
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

func TestBucketExistsContext_PropagatesCancel(t *testing.T) {
	t.Parallel()

	tb := testfake.NewFakeTB()
	done := make(chan struct{})

	go func() {
		defer close(done)
		s3assert.BucketExistsContext(tb, cancelledCtx(t), testCfg(), "any-bucket")
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("BucketExistsContext did not honor cancelled ctx within 2s")
	}

	if !tb.Errored() {
		t.Error("BucketExistsContext with cancelled ctx did not report Errorf")
	}
}
