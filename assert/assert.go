// Package assert provides post-apply assertion helpers for AWS resources.
// The zero-size struct pattern (var S3 s3Asserts) provides IDE-friendly
// namespace grouping without polluting the package namespace.
package assert

// S3 provides S3 bucket assertion methods.
var S3 s3Asserts

// DynamoDB provides DynamoDB table assertion methods.
var DynamoDB dynamoAsserts

// IAM provides IAM assertion methods. Pro-only methods call RequirePro internally.
var IAM iamAsserts

// SSM provides SSM Parameter Store assertion methods.
var SSM ssmAsserts

// Lambda provides Lambda function assertion methods.
var Lambda lambdaAsserts
