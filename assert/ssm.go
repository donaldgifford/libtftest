package assert

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"

	"github.com/donaldgifford/libtftest/awsx"
)

type ssmAsserts struct{}

// ParameterExists asserts that the named SSM parameter exists.
func (ssmAsserts) ParameterExists(tb testing.TB, cfg aws.Config, name string) {
	tb.Helper()

	client := awsx.NewSSM(cfg)
	_, err := client.GetParameter(context.Background(), &ssm.GetParameterInput{
		Name: aws.String(name),
	})
	if err != nil {
		tb.Errorf("ParameterExists(%q): %v", name, err)
	}
}

// ParameterHasValue asserts that the parameter has the expected value.
func (ssmAsserts) ParameterHasValue(tb testing.TB, cfg aws.Config, name, want string) {
	tb.Helper()

	client := awsx.NewSSM(cfg)
	out, err := client.GetParameter(context.Background(), &ssm.GetParameterInput{
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
