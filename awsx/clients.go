package awsx

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// NewS3 creates an S3 client with path-style addressing enabled.
func NewS3(cfg aws.Config) *s3.Client {
	return s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})
}

// NewDynamoDB creates a DynamoDB client.
func NewDynamoDB(cfg aws.Config) *dynamodb.Client {
	return dynamodb.NewFromConfig(cfg)
}

// NewIAM creates an IAM client.
func NewIAM(cfg aws.Config) *iam.Client {
	return iam.NewFromConfig(cfg)
}

// NewSSM creates an SSM client.
func NewSSM(cfg aws.Config) *ssm.Client {
	return ssm.NewFromConfig(cfg)
}

// NewSecrets creates a Secrets Manager client.
func NewSecrets(cfg aws.Config) *secretsmanager.Client {
	return secretsmanager.NewFromConfig(cfg)
}

// NewSQS creates an SQS client.
func NewSQS(cfg aws.Config) *sqs.Client {
	return sqs.NewFromConfig(cfg)
}

// NewSNS creates an SNS client.
func NewSNS(cfg aws.Config) *sns.Client {
	return sns.NewFromConfig(cfg)
}

// NewLambda creates a Lambda client.
func NewLambda(cfg aws.Config) *lambda.Client {
	return lambda.NewFromConfig(cfg)
}

// NewKMS creates a KMS client.
func NewKMS(cfg aws.Config) *kms.Client {
	return kms.NewFromConfig(cfg)
}

// NewKinesis creates a Kinesis client.
func NewKinesis(cfg aws.Config) *kinesis.Client {
	return kinesis.NewFromConfig(cfg)
}

// NewSTS creates an STS client.
func NewSTS(cfg aws.Config) *sts.Client {
	return sts.NewFromConfig(cfg)
}

// NewResourceGroupsTagging creates a Resource Groups Tagging API client.
func NewResourceGroupsTagging(cfg aws.Config) *resourcegroupstaggingapi.Client {
	return resourcegroupstaggingapi.NewFromConfig(cfg)
}
