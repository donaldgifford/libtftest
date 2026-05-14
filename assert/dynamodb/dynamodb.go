package dynamodb

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	ddbsdk "github.com/aws/aws-sdk-go-v2/service/dynamodb"

	"github.com/donaldgifford/libtftest/awsx"
)

// TableExistsContext asserts that the named DynamoDB table exists.
func TableExistsContext(tb testing.TB, ctx context.Context, cfg aws.Config, name string) {
	tb.Helper()

	client := awsx.NewDynamoDB(cfg)
	_, err := client.DescribeTable(ctx, &ddbsdk.DescribeTableInput{
		TableName: aws.String(name),
	})
	if err != nil {
		tb.Errorf("TableExists(%q): %v", name, err)
	}
}

// TableExists is a shim that calls TableExistsContext with tb.Context().
func TableExists(tb testing.TB, cfg aws.Config, name string) {
	tb.Helper()
	TableExistsContext(tb, tb.Context(), cfg, name)
}
