package lambda_test

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"

	lambdaassert "github.com/donaldgifford/libtftest/assert/lambda"
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

func TestFunctionExistsContext_PropagatesCancel(t *testing.T) {
	t.Parallel()

	tb := testfake.NewFakeTB()
	done := make(chan struct{})

	go func() {
		defer close(done)
		lambdaassert.FunctionExistsContext(tb, cancelledCtx(t), testCfg(), "any-fn")
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("FunctionExistsContext did not honor cancelled ctx within 2s")
	}

	if !tb.Errored() {
		t.Error("FunctionExistsContext with cancelled ctx did not report Errorf")
	}
}
