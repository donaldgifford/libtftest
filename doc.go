// Package libtftest wraps Terratest with opinionated,
// LocalStack-aware defaults for Terraform module integration
// testing.
//
// libtftest manages the LocalStack container lifecycle, injects
// provider and backend overrides into the workspace under test,
// provides pre-configured AWS SDK v2 clients via the awsx
// sub-package, and offers parallel-safe resource naming. The goal:
// module authors write ~10 lines of Go instead of ~200.
//
// # Typical use
//
//	func TestS3ModuleApply(t *testing.T) {
//		tc := libtftest.New(t, &libtftest.Options{
//			ModuleDir: "../",
//		})
//		tc.SetVar("bucket_name", tc.Prefix()+"-bucket")
//		tc.Apply()
//
//		s3assert.BucketExists(t, tc.AWS(), tc.Prefix()+"-bucket")
//	}
//
// # Package layout
//
// libtftest is organised as a small root package plus several
// sub-packages, each focused on a single concern:
//
//   - assert/<service> — post-apply assertions, one package per
//     AWS service (s3, dynamodb, iam, ssm, lambda, tags, snapshot)
//   - fixtures/<service> — pre-apply data seeding, one package per
//     AWS service (s3, ssm, secretsmanager, sqs)
//   - awsx — flat package of typed AWS SDK v2 client constructors
//   - harness — TestMain helpers for shared-container suites
//   - localstack — testcontainers-go wrapper for LocalStack
//   - tf — Terraform workspace management, override rendering
//   - sneakystack — LocalStack gap-filling HTTP proxy
//
// See DESIGN-0001 for the architectural overview and DESIGN-0003
// for the v0.2.0 layout refactor that introduced the per-service
// sub-packages.
package libtftest
