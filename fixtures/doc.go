// Package fixtures is deprecated. Use the per-service sub-packages
// instead — every Seed function that previously lived as
// fixtures.Seed<Service><Resource> moved to a package-level function
// in fixtures/<service>/ during the v0.2.0 layout refactor.
//
//	import s3fix "github.com/donaldgifford/libtftest/fixtures/s3"
//	s3fix.SeedObject(t, cfg, bucket, key, body)
//
// Migration map:
//
//	fixtures.SeedS3Object       ->  fixtures/s3.SeedObject
//	fixtures.SeedSSMParameter   ->  fixtures/ssm.SeedParameter
//	fixtures.SeedSecret         ->  fixtures/secretsmanager.SeedSecret
//	fixtures.SeedSQSMessage     ->  fixtures/sqs.SeedMessage
//
// The top-level package retains no exported surface and exists only
// as a deprecated discovery target. See DESIGN-0003 Part 1 for the
// layout rationale.
package fixtures
