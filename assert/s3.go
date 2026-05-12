package assert

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/donaldgifford/libtftest/awsx"
)

type s3Asserts struct{}

// BucketExistsContext is the ctx-aware variant of BucketExists.
func (s3Asserts) BucketExistsContext(tb testing.TB, ctx context.Context, cfg aws.Config, name string) {
	tb.Helper()

	client := awsx.NewS3(cfg)
	_, err := client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(name),
	})
	if err != nil {
		tb.Errorf("BucketExists(%q): %v", name, err)
	}
}

// BucketExists is a shim that calls BucketExistsContext with tb.Context().
func (s s3Asserts) BucketExists(tb testing.TB, cfg aws.Config, name string) {
	tb.Helper()
	s.BucketExistsContext(tb, tb.Context(), cfg, name)
}

// BucketHasEncryptionContext is the ctx-aware variant of BucketHasEncryption.
func (s3Asserts) BucketHasEncryptionContext(tb testing.TB, ctx context.Context, cfg aws.Config, name, algo string) {
	tb.Helper()

	client := awsx.NewS3(cfg)
	out, err := client.GetBucketEncryption(ctx, &s3.GetBucketEncryptionInput{
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
func (s s3Asserts) BucketHasEncryption(tb testing.TB, cfg aws.Config, name, algo string) {
	tb.Helper()
	s.BucketHasEncryptionContext(tb, tb.Context(), cfg, name, algo)
}

// BucketHasVersioningContext is the ctx-aware variant of BucketHasVersioning.
func (s3Asserts) BucketHasVersioningContext(tb testing.TB, ctx context.Context, cfg aws.Config, name string) {
	tb.Helper()

	client := awsx.NewS3(cfg)
	out, err := client.GetBucketVersioning(ctx, &s3.GetBucketVersioningInput{
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
func (s s3Asserts) BucketHasVersioning(tb testing.TB, cfg aws.Config, name string) {
	tb.Helper()
	s.BucketHasVersioningContext(tb, tb.Context(), cfg, name)
}

// BucketBlocksPublicAccessContext is the ctx-aware variant of BucketBlocksPublicAccess.
func (s3Asserts) BucketBlocksPublicAccessContext(tb testing.TB, ctx context.Context, cfg aws.Config, name string) {
	tb.Helper()

	client := awsx.NewS3(cfg)
	out, err := client.GetPublicAccessBlock(ctx, &s3.GetPublicAccessBlockInput{
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
func (s s3Asserts) BucketBlocksPublicAccess(tb testing.TB, cfg aws.Config, name string) {
	tb.Helper()
	s.BucketBlocksPublicAccessContext(tb, tb.Context(), cfg, name)
}

// BucketHasTagContext is the ctx-aware variant of BucketHasTag.
func (s3Asserts) BucketHasTagContext(tb testing.TB, ctx context.Context, cfg aws.Config, name, key, want string) {
	tb.Helper()

	client := awsx.NewS3(cfg)
	out, err := client.GetBucketTagging(ctx, &s3.GetBucketTaggingInput{
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
func (s s3Asserts) BucketHasTag(tb testing.TB, cfg aws.Config, name, key, want string) {
	tb.Helper()
	s.BucketHasTagContext(tb, tb.Context(), cfg, name, key, want)
}
