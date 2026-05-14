// Package secretsmanager provides pre-apply data-seeding fixtures
// for AWS Secrets Manager resources in LocalStack-backed Terraform
// module tests.
//
// Each Seed function registers a t.Cleanup that removes the fixture
// after the test. Cleanups use context.WithoutCancel(ctx) so they
// survive test-end cancellation. The cleanup path uses
// ForceDeleteWithoutRecovery=true so subsequent CreateSecret calls
// for the same name don't collide with the recovery-window state.
//
// Import alias convention: callers typically alias this package as
// secretsfix to coexist with the AWS SDK's secretsmanager package:
//
//	import (
//	    secretsfix "github.com/donaldgifford/libtftest/fixtures/secretsmanager"
//	    secretssdk "github.com/aws/aws-sdk-go-v2/service/secretsmanager"
//	)
package secretsmanager
