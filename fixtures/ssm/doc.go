// Package ssm provides pre-apply data-seeding fixtures for AWS
// Systems Manager (SSM) Parameter Store resources in LocalStack-
// backed Terraform module tests.
//
// Each Seed function registers a t.Cleanup that removes the fixture
// after the test. Cleanups use context.WithoutCancel(ctx) so they
// survive test-end cancellation.
//
// Import alias convention: callers typically alias this package as
// ssmfix to coexist with the AWS SDK's ssm package:
//
//	import (
//	    ssmfix "github.com/donaldgifford/libtftest/fixtures/ssm"
//	    ssmsdk "github.com/aws/aws-sdk-go-v2/service/ssm"
//	)
package ssm
