package assert

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	"github.com/donaldgifford/libtftest/awsx"
)

type dynamoAsserts struct{}

// TableExistsContext is the ctx-aware variant of TableExists.
func (dynamoAsserts) TableExistsContext(tb testing.TB, ctx context.Context, cfg aws.Config, name string) {
	tb.Helper()

	client := awsx.NewDynamoDB(cfg)
	_, err := client.DescribeTable(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(name),
	})
	if err != nil {
		tb.Errorf("TableExists(%q): %v", name, err)
	}
}

// TableExists is a shim that calls TableExistsContext with tb.Context().
func (d dynamoAsserts) TableExists(tb testing.TB, cfg aws.Config, name string) {
	tb.Helper()
	d.TableExistsContext(tb, tb.Context(), cfg, name)
}
