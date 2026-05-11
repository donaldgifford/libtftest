package assert

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"

	"github.com/donaldgifford/libtftest/awsx"
)

type lambdaAsserts struct{}

// FunctionExistsContext is the ctx-aware variant of FunctionExists.
func (lambdaAsserts) FunctionExistsContext(tb testing.TB, ctx context.Context, cfg aws.Config, name string) {
	tb.Helper()

	client := awsx.NewLambda(cfg)
	_, err := client.GetFunction(ctx, &lambda.GetFunctionInput{
		FunctionName: aws.String(name),
	})
	if err != nil {
		tb.Errorf("FunctionExists(%q): %v", name, err)
	}
}

// FunctionExists is a shim that calls FunctionExistsContext with tb.Context().
func (l lambdaAsserts) FunctionExists(tb testing.TB, cfg aws.Config, name string) {
	tb.Helper()
	l.FunctionExistsContext(tb, tb.Context(), cfg, name)
}
