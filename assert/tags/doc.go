// Package tags provides service-agnostic tag-propagation assertions
// backed by the AWS Resource Groups Tagging API.
//
// The flagship assertion is PropagatesFromRoot — given a baseline map
// of tag key/value pairs and a list of resource ARNs, it verifies that
// every key in the baseline is present (with the expected value) on
// every listed resource. Extra tags on the resources are allowed: this
// is a subset check, not an equality check. Failures are aggregated
// across resources so a single test run surfaces every missing or
// mismatched tag rather than stopping at the first error.
//
// All assertions follow the paired-method shape established in
// INV-0001: a context-aware variant ending in Context and a shim that
// calls the *Context variant with tb.Context(). The shim exists so
// module authors writing straight-line assertion code don't have to
// thread ctx through every call:
//
//	tagsassert.PropagatesFromRoot(t, cfg, baseline, arns...)
//	tagsassert.PropagatesFromRootContext(t, ctx, cfg, baseline, arns...)
//
// Import alias convention: callers typically alias this package as
// tagsassert to make the call site self-documenting and to coexist
// with any per-service assert package they're already using.
//
// # Why the Resource Groups Tagging API
//
// One AWS call regardless of resource type — no need to dispatch
// to per-service ListTagsForResource shapes. Works for any resource
// whose ARN the Tagging API can fetch, including resources the
// per-service SDK packages in awsx don't yet expose tag-listing
// helpers for.
//
// LocalStack OSS supports the Resource Groups Tagging API surface
// these assertions need (GetResources). The unit-level coverage uses
// internal/testfake to exercise the failure paths deterministically;
// the integration coverage in libtftest_integration_test.go verifies
// the round trip against LocalStack.
package tags
