package ssm

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	ssmsdk "github.com/aws/aws-sdk-go-v2/service/ssm"
	ssmtypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"

	"github.com/donaldgifford/libtftest/awsx"
)

// SeedParameterContext creates an SSM parameter and registers a
// cleanup (via context.WithoutCancel) to delete it after the test.
// When secure is true the parameter is created as a SecureString.
func SeedParameterContext(tb testing.TB, ctx context.Context, cfg aws.Config, name, value string, secure bool) {
	tb.Helper()

	client := awsx.NewSSM(cfg)

	paramType := ssmtypes.ParameterTypeString
	if secure {
		paramType = ssmtypes.ParameterTypeSecureString
	}

	_, err := client.PutParameter(ctx, &ssmsdk.PutParameterInput{
		Name:  aws.String(name),
		Value: aws.String(value),
		Type:  paramType,
	})
	if err != nil {
		tb.Fatalf("SeedParameter(%s): %v", name, err)
	}

	cleanupCtx := context.WithoutCancel(ctx)
	tb.Cleanup(func() {
		_, err := client.DeleteParameter(cleanupCtx, &ssmsdk.DeleteParameterInput{
			Name: aws.String(name),
		})
		if err != nil {
			tb.Errorf("cleanup SeedParameter(%s): %v", name, err)
		}
	})
}

// SeedParameter is a shim that calls SeedParameterContext with tb.Context().
func SeedParameter(tb testing.TB, cfg aws.Config, name, value string, secure bool) {
	tb.Helper()
	SeedParameterContext(tb, tb.Context(), cfg, name, value, secure)
}
