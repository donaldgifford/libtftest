package iam_test

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"

	iamassert "github.com/donaldgifford/libtftest/assert/iam"
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

func TestRoleExistsContext_PropagatesCancel(t *testing.T) {
	// RoleExistsContext calls RequirePro internally, which skips when
	// LOCALSTACK_AUTH_TOKEN is unset. Set it so we exercise the ctx
	// path. t.Setenv requires the test not be parallel.
	t.Setenv("LOCALSTACK_AUTH_TOKEN", "test-token")

	tb := testfake.NewFakeTB()
	done := make(chan struct{})

	go func() {
		defer close(done)
		iamassert.RoleExistsContext(tb, cancelledCtx(t), testCfg(), "any-role")
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("RoleExistsContext did not honor cancelled ctx within 2s")
	}

	if tb.Skipped() {
		t.Skip("RoleExistsContext was skipped by RequirePro — token env not honored?")
	}
	if !tb.Errored() {
		t.Error("RoleExistsContext with cancelled ctx did not report Errorf")
	}
}
