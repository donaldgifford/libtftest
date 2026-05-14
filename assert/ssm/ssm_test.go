package ssm_test

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"

	ssmassert "github.com/donaldgifford/libtftest/assert/ssm"
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

func TestParameterExistsContext_PropagatesCancel(t *testing.T) {
	t.Parallel()

	tb := testfake.NewFakeTB()
	done := make(chan struct{})

	go func() {
		defer close(done)
		ssmassert.ParameterExistsContext(tb, cancelledCtx(t), testCfg(), "any-param")
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("ParameterExistsContext did not honor cancelled ctx within 2s")
	}

	if !tb.Errored() {
		t.Error("ParameterExistsContext with cancelled ctx did not report Errorf")
	}
}
