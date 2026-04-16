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

// BucketExists asserts that the named S3 bucket exists.
func (s3Asserts) BucketExists(tb testing.TB, cfg aws.Config, name string) {
	tb.Helper()

	client := awsx.NewS3(cfg)
	_, err := client.HeadBucket(context.Background(), &s3.HeadBucketInput{
		Bucket: aws.String(name),
	})
	if err != nil {
		tb.Errorf("BucketExists(%q): %v", name, err)
	}
}

// BucketHasEncryption asserts that the bucket has the specified encryption algorithm.
func (s3Asserts) BucketHasEncryption(tb testing.TB, cfg aws.Config, name, algo string) {
	tb.Helper()

	client := awsx.NewS3(cfg)
	out, err := client.GetBucketEncryption(context.Background(), &s3.GetBucketEncryptionInput{
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

// BucketHasVersioning asserts that the bucket has versioning enabled.
func (s3Asserts) BucketHasVersioning(tb testing.TB, cfg aws.Config, name string) {
	tb.Helper()

	client := awsx.NewS3(cfg)
	out, err := client.GetBucketVersioning(context.Background(), &s3.GetBucketVersioningInput{
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

// BucketBlocksPublicAccess asserts that the bucket blocks all public access.
func (s3Asserts) BucketBlocksPublicAccess(tb testing.TB, cfg aws.Config, name string) {
	tb.Helper()

	client := awsx.NewS3(cfg)
	out, err := client.GetPublicAccessBlock(context.Background(), &s3.GetPublicAccessBlockInput{
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

// BucketHasTag asserts that the bucket has a tag with the given key and value.
func (s3Asserts) BucketHasTag(tb testing.TB, cfg aws.Config, name, key, want string) {
	tb.Helper()

	client := awsx.NewS3(cfg)
	out, err := client.GetBucketTagging(context.Background(), &s3.GetBucketTaggingInput{
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
