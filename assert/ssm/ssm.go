package ssm

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	ssmsdk "github.com/aws/aws-sdk-go-v2/service/ssm"

	"github.com/donaldgifford/libtftest/awsx"
)

// ParameterExistsContext asserts that the named SSM parameter exists.
func ParameterExistsContext(tb testing.TB, ctx context.Context, cfg aws.Config, name string) {
	tb.Helper()

	client := awsx.NewSSM(cfg)
	_, err := client.GetParameter(ctx, &ssmsdk.GetParameterInput{
		Name: aws.String(name),
	})
	if err != nil {
		tb.Errorf("ParameterExists(%q): %v", name, err)
	}
}

// ParameterExists is a shim that calls ParameterExistsContext with tb.Context().
func ParameterExists(tb testing.TB, cfg aws.Config, name string) {
	tb.Helper()
	ParameterExistsContext(tb, tb.Context(), cfg, name)
}

// ParameterHasValueContext asserts that the named SSM parameter
// resolves to want under decryption.
func ParameterHasValueContext(tb testing.TB, ctx context.Context, cfg aws.Config, name, want string) {
	tb.Helper()

	client := awsx.NewSSM(cfg)
	out, err := client.GetParameter(ctx, &ssmsdk.GetParameterInput{
		Name:           aws.String(name),
		WithDecryption: aws.Bool(true),
	})
	if err != nil {
		tb.Errorf("ParameterHasValue(%q): %v", name, err)
		return
	}

	if got := aws.ToString(out.Parameter.Value); got != want {
		tb.Errorf("ParameterHasValue(%q) = %q, want %q", name, got, want)
	}
}

// ParameterHasValue is a shim that calls ParameterHasValueContext with tb.Context().
func ParameterHasValue(tb testing.TB, cfg aws.Config, name, want string) {
	tb.Helper()
	ParameterHasValueContext(tb, tb.Context(), cfg, name, want)
}
