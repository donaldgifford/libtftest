// Package s3 provides post-apply assertions for AWS S3 resources
// created by Terraform modules under test.
//
// All assertions follow the paired-method shape established in
// INV-0001: a context-aware variant ending in Context and a shim
// that calls the *Context variant with tb.Context(). The shim
// exists so module authors writing straight-line assertion code
// don't have to thread ctx through every line:
//
//	s3assert.BucketExists(t, cfg, name)
//	s3assert.BucketExistsContext(t, ctx, cfg, name)
//
// Every *Context assertion respects ctx cancellation — a cancelled
// ctx surfaces as a tb.Errorf call rather than a panic or a hang.
//
// Import alias convention: callers typically alias this package as
// s3assert to avoid a naming collision with the AWS SDK's s3
// package:
//
//	import (
//	    s3assert "github.com/donaldgifford/libtftest/assert/s3"
//	    s3sdk   "github.com/aws/aws-sdk-go-v2/service/s3"
//	)
package s3
