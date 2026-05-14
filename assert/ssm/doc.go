// Package ssm provides post-apply assertions for AWS Systems Manager
// (SSM) Parameter Store resources created by Terraform modules under
// test.
//
// All assertions follow the paired-method shape from INV-0001: a
// context-aware variant ending in Context and a shim that calls the
// *Context variant with tb.Context().
//
// Import alias convention: callers typically alias this package as
// ssmassert to coexist with the AWS SDK's ssm package:
//
//	import (
//	    ssmassert "github.com/donaldgifford/libtftest/assert/ssm"
//	    ssmsdk   "github.com/aws/aws-sdk-go-v2/service/ssm"
//	)
package ssm
