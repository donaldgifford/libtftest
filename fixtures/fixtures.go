// Package fixtures provides pre-apply data seeding functions for LocalStack resources.
// Each Seed function registers a t.Cleanup that removes the fixture.
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

// SeedS3Object uploads an object to the given S3 bucket and registers
// cleanup to delete it.
func SeedS3Object(tb testing.TB, cfg aws.Config, bucket, key string, body []byte) {
	tb.Helper()

	client := awsx.NewS3(cfg)
	ctx := context.Background()

	_, err := client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(body),
	})
	if err != nil {
		tb.Fatalf("SeedS3Object(%s/%s): %v", bucket, key, err)
	}

	tb.Cleanup(func() {
		_, err := client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		})
		if err != nil {
			tb.Errorf("cleanup SeedS3Object(%s/%s): %v", bucket, key, err)
		}
	})
}

// SeedSSMParameter creates an SSM parameter and registers cleanup to delete it.
func SeedSSMParameter(tb testing.TB, cfg aws.Config, name, value string, secure bool) {
	tb.Helper()

	client := awsx.NewSSM(cfg)
	ctx := context.Background()

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

	tb.Cleanup(func() {
		_, err := client.DeleteParameter(ctx, &ssm.DeleteParameterInput{
			Name: aws.String(name),
		})
		if err != nil {
			tb.Errorf("cleanup SeedSSMParameter(%s): %v", name, err)
		}
	})
}

// SeedSecret creates a Secrets Manager secret and registers cleanup.
func SeedSecret(tb testing.TB, cfg aws.Config, name, value string) {
	tb.Helper()

	client := awsx.NewSecrets(cfg)
	ctx := context.Background()

	_, err := client.CreateSecret(ctx, &secretsmanager.CreateSecretInput{
		Name:         aws.String(name),
		SecretString: aws.String(value),
	})
	if err != nil {
		tb.Fatalf("SeedSecret(%s): %v", name, err)
	}

	tb.Cleanup(func() {
		_, err := client.DeleteSecret(ctx, &secretsmanager.DeleteSecretInput{
			SecretId:                   aws.String(name),
			ForceDeleteWithoutRecovery: aws.Bool(true),
		})
		if err != nil {
			tb.Errorf("cleanup SeedSecret(%s): %v", name, err)
		}
	})
}

// SeedSQSMessage sends a message to the given SQS queue URL.
// No cleanup is registered — messages are consumed by the test.
func SeedSQSMessage(tb testing.TB, cfg aws.Config, queueURL, body string) {
	tb.Helper()

	client := awsx.NewSQS(cfg)

	_, err := client.SendMessage(context.Background(), &sqs.SendMessageInput{
		QueueUrl:    aws.String(queueURL),
		MessageBody: aws.String(body),
	})
	if err != nil {
		tb.Fatalf("SeedSQSMessage(%s): %v", queueURL, err)
	}
}
