// Package iam provides post-apply assertions for AWS IAM resources
// created by Terraform modules under test.
//
// Every assertion in this package gates on [libtftest.RequirePro]
// because LocalStack OSS does not implement the IAM resource-level
// APIs (GetRole, GetRolePolicy, etc.) — they require LocalStack Pro.
// Pro-only functions carry the `// libtftest:requires pro <reason>`
// marker for the docgen feature-matrix tool.
//
// All assertions follow the paired-method shape from INV-0001: a
// context-aware variant ending in Context and a shim that calls the
// *Context variant with tb.Context().
//
// Import alias convention: callers typically alias this package as
// iamassert to coexist with the AWS SDK's iam package:
//
//	import (
//	    iamassert "github.com/donaldgifford/libtftest/assert/iam"
//	    iamsdk   "github.com/aws/aws-sdk-go-v2/service/iam"
//	)
package iam
