// Package awsx provides AWS SDK v2 client constructors configured
// for LocalStack-backed Terraform module tests.
//
// Every constructor follows the same minimal shape:
//
//	func New<Service>(cfg aws.Config) *<service>.Client
//
// The constructor returns a client that respects the path-style
// addressing and force-path-style settings required by LocalStack,
// and that uses the credentials and endpoint baked into cfg by
// [libtftest.TestCase.AWS] without further mutation.
//
// # Deliberate flat layout
//
// Unlike the assert/ and fixtures/ packages — which adopted a
// per-service sub-package layout in v0.2.0 — awsx stays as a single
// flat package on purpose:
//
//   - Each constructor is ~10 lines with no per-service surface
//     beyond the AWS SDK's own client type
//   - There is no namespacing payoff: callers write awsx.NewS3(cfg)
//     either way; the AWS SDK's own s3 package is what dominates
//     the call site
//   - Splitting awsx/s3/, awsx/dynamodb/, etc. would generate one
//     two-function package per service with no shared surface
//
// DESIGN-0003 Resolved Question 1 captures the call. If a future
// need arises to expose service-specific helpers beyond the
// constructor, revisit then.
package awsx
