// Package lambda provides post-apply assertions for AWS Lambda
// resources created by Terraform modules under test.
//
// All assertions follow the paired-method shape from INV-0001: a
// context-aware variant ending in Context and a shim that calls the
// *Context variant with tb.Context().
//
// Import alias convention: callers typically alias this package as
// lambdaassert to coexist with the AWS SDK's lambda package:
//
//	import (
//	    lambdaassert "github.com/donaldgifford/libtftest/assert/lambda"
//	    lambdasdk   "github.com/aws/aws-sdk-go-v2/service/lambda"
//	)
package lambda
