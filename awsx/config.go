package awsx

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

// New returns an aws.Config whose BaseEndpoint routes all services to the
// given LocalStack edge URL with dummy credentials.
func New(ctx context.Context, edgeURL string) (aws.Config, error) {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithBaseEndpoint(edgeURL),
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider("test", "test", ""),
		),
	)
	if err != nil {
		return aws.Config{}, fmt.Errorf("load aws config: %w", err)
	}

	return cfg, nil
}
