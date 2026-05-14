package s3

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	s3sdk "github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/donaldgifford/libtftest/awsx"
)

// BucketExistsContext asserts that the named S3 bucket exists.
func BucketExistsContext(tb testing.TB, ctx context.Context, cfg aws.Config, name string) {
	tb.Helper()

	client := awsx.NewS3(cfg)
	_, err := client.HeadBucket(ctx, &s3sdk.HeadBucketInput{
		Bucket: aws.String(name),
	})
	if err != nil {
		tb.Errorf("BucketExists(%q): %v", name, err)
	}
}

// BucketExists is a shim that calls BucketExistsContext with tb.Context().
func BucketExists(tb testing.TB, cfg aws.Config, name string) {
	tb.Helper()
	BucketExistsContext(tb, tb.Context(), cfg, name)
}

// BucketHasEncryptionContext asserts that the named S3 bucket has
// server-side encryption configured with the given algorithm.
func BucketHasEncryptionContext(tb testing.TB, ctx context.Context, cfg aws.Config, name, algo string) {
	tb.Helper()

	client := awsx.NewS3(cfg)
	out, err := client.GetBucketEncryption(ctx, &s3sdk.GetBucketEncryptionInput{
		Bucket: aws.String(name),
	})
	if err != nil {
		tb.Errorf("BucketHasEncryption(%q, %q): %v", name, algo, err)
		return
	}

	if out.ServerSideEncryptionConfiguration == nil {
		tb.Errorf("BucketHasEncryption(%q): no encryption configuration", name)
		return
	}

	for _, rule := range out.ServerSideEncryptionConfiguration.Rules {
		if rule.ApplyServerSideEncryptionByDefault != nil {
			if string(rule.ApplyServerSideEncryptionByDefault.SSEAlgorithm) == algo {
				return
			}
		}
	}

	tb.Errorf("BucketHasEncryption(%q, %q): algorithm not found", name, algo)
}

// BucketHasEncryption is a shim that calls BucketHasEncryptionContext with tb.Context().
func BucketHasEncryption(tb testing.TB, cfg aws.Config, name, algo string) {
	tb.Helper()
	BucketHasEncryptionContext(tb, tb.Context(), cfg, name, algo)
}

// BucketHasVersioningContext asserts that the named S3 bucket has
// versioning Enabled.
func BucketHasVersioningContext(tb testing.TB, ctx context.Context, cfg aws.Config, name string) {
	tb.Helper()

	client := awsx.NewS3(cfg)
	out, err := client.GetBucketVersioning(ctx, &s3sdk.GetBucketVersioningInput{
		Bucket: aws.String(name),
	})
	if err != nil {
		tb.Errorf("BucketHasVersioning(%q): %v", name, err)
		return
	}

	if out.Status != s3types.BucketVersioningStatusEnabled {
		tb.Errorf("BucketHasVersioning(%q): status = %q, want Enabled", name, out.Status)
	}
}

// BucketHasVersioning is a shim that calls BucketHasVersioningContext with tb.Context().
func BucketHasVersioning(tb testing.TB, cfg aws.Config, name string) {
	tb.Helper()
	BucketHasVersioningContext(tb, tb.Context(), cfg, name)
}

// BucketBlocksPublicAccessContext asserts that the named S3 bucket
// has all four public-access-block flags set to true.
func BucketBlocksPublicAccessContext(tb testing.TB, ctx context.Context, cfg aws.Config, name string) {
	tb.Helper()

	client := awsx.NewS3(cfg)
	out, err := client.GetPublicAccessBlock(ctx, &s3sdk.GetPublicAccessBlockInput{
		Bucket: aws.String(name),
	})
	if err != nil {
		tb.Errorf("BucketBlocksPublicAccess(%q): %v", name, err)
		return
	}

	pac := out.PublicAccessBlockConfiguration
	if pac == nil {
		tb.Errorf("BucketBlocksPublicAccess(%q): no public access block config", name)
		return
	}

	if !aws.ToBool(pac.BlockPublicAcls) {
		tb.Errorf("BucketBlocksPublicAccess(%q): BlockPublicAcls = false", name)
	}
	if !aws.ToBool(pac.BlockPublicPolicy) {
		tb.Errorf("BucketBlocksPublicAccess(%q): BlockPublicPolicy = false", name)
	}
	if !aws.ToBool(pac.IgnorePublicAcls) {
		tb.Errorf("BucketBlocksPublicAccess(%q): IgnorePublicAcls = false", name)
	}
	if !aws.ToBool(pac.RestrictPublicBuckets) {
		tb.Errorf("BucketBlocksPublicAccess(%q): RestrictPublicBuckets = false", name)
	}
}

// BucketBlocksPublicAccess is a shim that calls BucketBlocksPublicAccessContext with tb.Context().
func BucketBlocksPublicAccess(tb testing.TB, cfg aws.Config, name string) {
	tb.Helper()
	BucketBlocksPublicAccessContext(tb, tb.Context(), cfg, name)
}

// BucketHasTagContext asserts that the named S3 bucket carries the
// given tag key with the given value.
func BucketHasTagContext(tb testing.TB, ctx context.Context, cfg aws.Config, name, key, want string) {
	tb.Helper()

	client := awsx.NewS3(cfg)
	out, err := client.GetBucketTagging(ctx, &s3sdk.GetBucketTaggingInput{
		Bucket: aws.String(name),
	})
	if err != nil {
		tb.Errorf("BucketHasTag(%q, %q): %v", name, key, err)
		return
	}

	for _, tag := range out.TagSet {
		if aws.ToString(tag.Key) == key {
			if got := aws.ToString(tag.Value); got != want {
				tb.Errorf("BucketHasTag(%q, %q) = %q, want %q", name, key, got, want)
			}
			return
		}
	}

	tb.Errorf("BucketHasTag(%q, %q): tag not found", name, key)
}

// BucketHasTag is a shim that calls BucketHasTagContext with tb.Context().
func BucketHasTag(tb testing.TB, cfg aws.Config, name, key, want string) {
	tb.Helper()
	BucketHasTagContext(tb, tb.Context(), cfg, name, key, want)
}
