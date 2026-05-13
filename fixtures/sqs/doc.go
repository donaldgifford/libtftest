// Package sqs provides pre-apply data-seeding fixtures for AWS SQS
// resources in LocalStack-backed Terraform module tests.
//
// SeedMessage sends a single message to a queue and does NOT register
// a cleanup — SQS messages are consumed by the test itself, not by
// a teardown handler.
//
// Import alias convention: callers typically alias this package as
// sqsfix to coexist with the AWS SDK's sqs package:
//
//	import (
//	    sqsfix "github.com/donaldgifford/libtftest/fixtures/sqs"
//	    sqssdk "github.com/aws/aws-sdk-go-v2/service/sqs"
//	)
package sqs
