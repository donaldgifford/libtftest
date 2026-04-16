package assert

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	"github.com/donaldgifford/libtftest/awsx"
)

type dynamoAsserts struct{}

// TableExists asserts that the named DynamoDB table exists.
func (dynamoAsserts) TableExists(tb testing.TB, cfg aws.Config, name string) {
	tb.Helper()

	client := awsx.NewDynamoDB(cfg)
	_, err := client.DescribeTable(context.Background(), &dynamodb.DescribeTableInput{
		TableName: aws.String(name),
	})
	if err != nil {
		tb.Errorf("TableExists(%q): %v", name, err)
	}
}
