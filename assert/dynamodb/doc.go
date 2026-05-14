// Package dynamodb provides post-apply assertions for AWS DynamoDB
// resources created by Terraform modules under test.
//
// All assertions follow the paired-method shape from INV-0001: a
// context-aware variant ending in Context and a shim that calls the
// *Context variant with tb.Context().
//
// Import alias convention: callers typically alias this package as
// ddbassert to coexist with the AWS SDK's dynamodb package:
//
//	import (
//	    ddbassert "github.com/donaldgifford/libtftest/assert/dynamodb"
//	    ddbsdk   "github.com/aws/aws-sdk-go-v2/service/dynamodb"
//	)
package dynamodb
