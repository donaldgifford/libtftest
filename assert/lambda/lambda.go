package lambda

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	lambdasdk "github.com/aws/aws-sdk-go-v2/service/lambda"

	"github.com/donaldgifford/libtftest/awsx"
)

// FunctionExistsContext asserts that the named Lambda function exists.
func FunctionExistsContext(tb testing.TB, ctx context.Context, cfg aws.Config, name string) {
	tb.Helper()

	client := awsx.NewLambda(cfg)
	_, err := client.GetFunction(ctx, &lambdasdk.GetFunctionInput{
		FunctionName: aws.String(name),
	})
	if err != nil {
		tb.Errorf("FunctionExists(%q): %v", name, err)
	}
}

// FunctionExists is a shim that calls FunctionExistsContext with tb.Context().
func FunctionExists(tb testing.TB, cfg aws.Config, name string) {
	tb.Helper()
	FunctionExistsContext(tb, tb.Context(), cfg, name)
}
