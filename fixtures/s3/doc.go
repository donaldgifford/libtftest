// Package s3 provides pre-apply data-seeding fixtures for AWS S3
// resources in LocalStack-backed Terraform module tests.
//
// Each Seed function registers a t.Cleanup that removes the fixture
// after the test. Cleanups use context.WithoutCancel(ctx) so they
// survive test-end cancellation — the cleanup path runs even when
// the test's own context has been cancelled by t.Failed or a
// parent's deadline.
//
// All fixtures follow the paired-method shape from INV-0001: a
// context-aware variant ending in Context and a shim that calls
// the *Context variant with tb.Context().
//
// Import alias convention: callers typically alias this package as
// s3fix to coexist with the AWS SDK's s3 package:
//
//	import (
//	    s3fix "github.com/donaldgifford/libtftest/fixtures/s3"
//	    s3sdk "github.com/aws/aws-sdk-go-v2/service/s3"
//	)
package s3
