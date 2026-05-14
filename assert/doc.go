// Package assert is deprecated. Use the per-service sub-packages
// instead — every assertion that previously lived under
// assert.<Service>.<Method> moved to a package-level function in
// assert/<service>/ during the v0.2.0 layout refactor.
//
//	import s3assert "github.com/donaldgifford/libtftest/assert/s3"
//	s3assert.BucketExists(t, cfg, name)
//
// Migration map:
//
//	assert.S3        ->  github.com/donaldgifford/libtftest/assert/s3
//	assert.DynamoDB  ->  github.com/donaldgifford/libtftest/assert/dynamodb
//	assert.IAM       ->  github.com/donaldgifford/libtftest/assert/iam (Pro)
//	assert.SSM       ->  github.com/donaldgifford/libtftest/assert/ssm
//	assert.Lambda    ->  github.com/donaldgifford/libtftest/assert/lambda
//
// The top-level package retains no exported surface and exists only
// as a deprecated discovery target. See DESIGN-0003 Part 1 for the
// layout rationale and INV-0002 for the original EKS-scale analysis
// that drove the refactor.
package assert
