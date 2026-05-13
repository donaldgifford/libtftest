package sqs

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	sqssdk "github.com/aws/aws-sdk-go-v2/service/sqs"

	"github.com/donaldgifford/libtftest/awsx"
)

// SeedMessageContext sends a message to the given SQS queue URL. No
// cleanup is registered — SQS messages are consumed by the test
// itself, not by a teardown handler.
func SeedMessageContext(tb testing.TB, ctx context.Context, cfg aws.Config, queueURL, body string) {
	tb.Helper()

	client := awsx.NewSQS(cfg)

	_, err := client.SendMessage(ctx, &sqssdk.SendMessageInput{
		QueueUrl:    aws.String(queueURL),
		MessageBody: aws.String(body),
	})
	if err != nil {
		tb.Fatalf("SeedMessage(%s): %v", queueURL, err)
	}
}

// SeedMessage is a shim that calls SeedMessageContext with tb.Context().
func SeedMessage(tb testing.TB, cfg aws.Config, queueURL, body string) {
	tb.Helper()
	SeedMessageContext(tb, tb.Context(), cfg, queueURL, body)
}
