package s3

import (
	"bytes"
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	s3sdk "github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/donaldgifford/libtftest/awsx"
)

// SeedObjectContext uploads an object to the given S3 bucket and
// registers a cleanup (via context.WithoutCancel) to delete it after
// the test.
func SeedObjectContext(tb testing.TB, ctx context.Context, cfg aws.Config, bucket, key string, body []byte) {
	tb.Helper()

	client := awsx.NewS3(cfg)

	_, err := client.PutObject(ctx, &s3sdk.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(body),
	})
	if err != nil {
		tb.Fatalf("SeedObject(%s/%s): %v", bucket, key, err)
	}

	cleanupCtx := context.WithoutCancel(ctx)
	tb.Cleanup(func() {
		_, err := client.DeleteObject(cleanupCtx, &s3sdk.DeleteObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		})
		if err != nil {
			tb.Errorf("cleanup SeedObject(%s/%s): %v", bucket, key, err)
		}
	})
}

// SeedObject is a shim that calls SeedObjectContext with tb.Context().
func SeedObject(tb testing.TB, cfg aws.Config, bucket, key string, body []byte) {
	tb.Helper()
	SeedObjectContext(tb, tb.Context(), cfg, bucket, key, body)
}
