package assert

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"

	"github.com/donaldgifford/libtftest/awsx"
)

type ssmAsserts struct{}

// ParameterExistsContext is the ctx-aware variant of ParameterExists.
func (ssmAsserts) ParameterExistsContext(tb testing.TB, ctx context.Context, cfg aws.Config, name string) {
	tb.Helper()

	client := awsx.NewSSM(cfg)
	_, err := client.GetParameter(ctx, &ssm.GetParameterInput{
		Name: aws.String(name),
	})
	if err != nil {
		tb.Errorf("ParameterExists(%q): %v", name, err)
	}
}

// ParameterExists is a shim that calls ParameterExistsContext with tb.Context().
func (s ssmAsserts) ParameterExists(tb testing.TB, cfg aws.Config, name string) {
	tb.Helper()
	s.ParameterExistsContext(tb, tb.Context(), cfg, name)
}

// ParameterHasValueContext is the ctx-aware variant of ParameterHasValue.
func (ssmAsserts) ParameterHasValueContext(tb testing.TB, ctx context.Context, cfg aws.Config, name, want string) {
	tb.Helper()

	client := awsx.NewSSM(cfg)
	out, err := client.GetParameter(ctx, &ssm.GetParameterInput{
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
func (s ssmAsserts) ParameterHasValue(tb testing.TB, cfg aws.Config, name, want string) {
	tb.Helper()
	s.ParameterHasValueContext(tb, tb.Context(), cfg, name, want)
}
