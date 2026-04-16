package assert

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"

	"github.com/donaldgifford/libtftest/awsx"
)

type lambdaAsserts struct{}

// FunctionExists asserts that the named Lambda function exists.
func (lambdaAsserts) FunctionExists(tb testing.TB, cfg aws.Config, name string) {
	tb.Helper()

	client := awsx.NewLambda(cfg)
	_, err := client.GetFunction(context.Background(), &lambda.GetFunctionInput{
		FunctionName: aws.String(name),
	})
	if err != nil {
		tb.Errorf("FunctionExists(%q): %v", name, err)
	}
}
