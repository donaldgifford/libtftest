package secretsmanager

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	secretssdk "github.com/aws/aws-sdk-go-v2/service/secretsmanager"

	"github.com/donaldgifford/libtftest/awsx"
)

// SeedSecretContext creates a Secrets Manager secret and registers a
// cleanup (via context.WithoutCancel) to delete it after the test.
// ForceDeleteWithoutRecovery is set so subsequent CreateSecret calls
// for the same name don't collide with the recovery-window state.
func SeedSecretContext(tb testing.TB, ctx context.Context, cfg aws.Config, name, value string) {
	tb.Helper()

	client := awsx.NewSecrets(cfg)

	_, err := client.CreateSecret(ctx, &secretssdk.CreateSecretInput{
		Name:         aws.String(name),
		SecretString: aws.String(value),
	})
	if err != nil {
		tb.Fatalf("SeedSecret(%s): %v", name, err)
	}

	cleanupCtx := context.WithoutCancel(ctx)
	tb.Cleanup(func() {
		_, err := client.DeleteSecret(cleanupCtx, &secretssdk.DeleteSecretInput{
			SecretId:                   aws.String(name),
			ForceDeleteWithoutRecovery: aws.Bool(true),
		})
		if err != nil {
			tb.Errorf("cleanup SeedSecret(%s): %v", name, err)
		}
	})
}

// SeedSecret is a shim that calls SeedSecretContext with tb.Context().
func SeedSecret(tb testing.TB, cfg aws.Config, name, value string) {
	tb.Helper()
	SeedSecretContext(tb, tb.Context(), cfg, name, value)
}
