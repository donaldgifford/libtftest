// Package fixtures provides pre-apply data seeding functions for LocalStack resources.
// Each Seed function registers a t.Cleanup that removes the fixture.
//
// Cleanup callbacks use context.WithoutCancel(ctx) so they survive test-end
// cancellation. The passing case is semantically identical to using
// tb.Context() directly; the failing/cancelled case is the one that matters.
package fixtures

import (
	"bytes"
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	ssmtypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"

	"github.com/donaldgifford/libtftest/awsx"
)

// SeedS3ObjectContext uploads an object to the given S3 bucket and registers
// cleanup (via context.WithoutCancel) to delete it after the test.
func SeedS3ObjectContext(tb testing.TB, ctx context.Context, cfg aws.Config, bucket, key string, body []byte) {
	tb.Helper()

	client := awsx.NewS3(cfg)

	_, err := client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(body),
	})
	if err != nil {
		tb.Fatalf("SeedS3Object(%s/%s): %v", bucket, key, err)
	}

	cleanupCtx := context.WithoutCancel(ctx)
	tb.Cleanup(func() {
		_, err := client.DeleteObject(cleanupCtx, &s3.DeleteObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		})
		if err != nil {
			tb.Errorf("cleanup SeedS3Object(%s/%s): %v", bucket, key, err)
		}
	})
}

// SeedS3Object is a shim that calls SeedS3ObjectContext with tb.Context().
func SeedS3Object(tb testing.TB, cfg aws.Config, bucket, key string, body []byte) {
	tb.Helper()
	SeedS3ObjectContext(tb, tb.Context(), cfg, bucket, key, body)
}

// SeedSSMParameterContext creates an SSM parameter and registers cleanup
// (via context.WithoutCancel) to delete it after the test.
func SeedSSMParameterContext(tb testing.TB, ctx context.Context, cfg aws.Config, name, value string, secure bool) {
	tb.Helper()

	client := awsx.NewSSM(cfg)

	paramType := ssmtypes.ParameterTypeString
	if secure {
		paramType = ssmtypes.ParameterTypeSecureString
	}

	_, err := client.PutParameter(ctx, &ssm.PutParameterInput{
		Name:  aws.String(name),
		Value: aws.String(value),
		Type:  paramType,
	})
	if err != nil {
		tb.Fatalf("SeedSSMParameter(%s): %v", name, err)
	}

	cleanupCtx := context.WithoutCancel(ctx)
	tb.Cleanup(func() {
		_, err := client.DeleteParameter(cleanupCtx, &ssm.DeleteParameterInput{
			Name: aws.String(name),
		})
		if err != nil {
			tb.Errorf("cleanup SeedSSMParameter(%s): %v", name, err)
		}
	})
}

// SeedSSMParameter is a shim that calls SeedSSMParameterContext with tb.Context().
func SeedSSMParameter(tb testing.TB, cfg aws.Config, name, value string, secure bool) {
	tb.Helper()
	SeedSSMParameterContext(tb, tb.Context(), cfg, name, value, secure)
}

// SeedSecretContext creates a Secrets Manager secret and registers cleanup
// (via context.WithoutCancel) to delete it after the test.
func SeedSecretContext(tb testing.TB, ctx context.Context, cfg aws.Config, name, value string) {
	tb.Helper()

	client := awsx.NewSecrets(cfg)

	_, err := client.CreateSecret(ctx, &secretsmanager.CreateSecretInput{
		Name:         aws.String(name),
		SecretString: aws.String(value),
	})
	if err != nil {
		tb.Fatalf("SeedSecret(%s): %v", name, err)
	}

	cleanupCtx := context.WithoutCancel(ctx)
	tb.Cleanup(func() {
		_, err := client.DeleteSecret(cleanupCtx, &secretsmanager.DeleteSecretInput{
			SecretId:                   aws.String(name),
			ForceDeleteWithoutRecovery: aws.Bool(true),
		})
		if err != nil {
			tb.Errorf("cleanup SeedSecret(%s): %v", name, err)
		}
	})
}

// SeedSecret is a shim that calls SeedSecretContext with tb.Context().
func SeedSecret(tb testing.TB, cfg aws.Config, name, value string) {
	tb.Helper()
	SeedSecretContext(tb, tb.Context(), cfg, name, value)
}

// SeedSQSMessageContext sends a message to the given SQS queue URL.
// No cleanup is registered — messages are consumed by the test.
func SeedSQSMessageContext(tb testing.TB, ctx context.Context, cfg aws.Config, queueURL, body string) {
	tb.Helper()

	client := awsx.NewSQS(cfg)

	_, err := client.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    aws.String(queueURL),
		MessageBody: aws.String(body),
	})
	if err != nil {
		tb.Fatalf("SeedSQSMessage(%s): %v", queueURL, err)
	}
}

// SeedSQSMessage is a shim that calls SeedSQSMessageContext with tb.Context().
func SeedSQSMessage(tb testing.TB, cfg aws.Config, queueURL, body string) {
	tb.Helper()
	SeedSQSMessageContext(tb, tb.Context(), cfg, queueURL, body)
}
