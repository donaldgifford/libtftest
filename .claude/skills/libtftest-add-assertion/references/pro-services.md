# LocalStack Pro-only services

Reference list for `libtftest:add-assertion` to decide whether to insert
`libtftest.RequirePro(tb)` in a generated method.

This list reflects LocalStack 4.x. Always cross-check with the upstream
LocalStack feature-coverage matrix before committing — it changes between
minor versions.

## Known Pro-only (RequirePro should be inserted)

- IAM **enforcement** (existence checks may work on Community; actual policy
  evaluation requires Pro)
- IAM Identity Center / SSO Admin
- Organizations
- Control Tower
- AppSync (advanced features)
- ECS task placement strategies (basic ECS works on Community)
- EKS (Pro-only as of 4.4)
- RDS (Pro-only — Community has limited surrogates)
- ElasticSearch / OpenSearch (Pro)
- DocumentDB (Pro)
- Glue (Pro)
- Athena (Pro)
- Redshift (Pro)

## Community-supported (no RequirePro)

- S3
- DynamoDB
- SQS
- SNS
- Lambda
- API Gateway (basic)
- KMS (most operations)
- SSM Parameter Store
- Secrets Manager
- CloudWatch Logs (basic)
- Kinesis
- EventBridge (basic)
- STS
- Step Functions

## Mixed / depends on the assertion

Some services have a mix of Community and Pro behavior. Default to inserting
`RequirePro` if the assertion exercises:

- Cross-account access
- Resource policies that require evaluation
- Encryption-at-rest with customer-managed keys (KMS interaction matters)

If unsure, ask the user explicitly: "Does this assertion need real AWS IAM
enforcement to be meaningful?" If yes → Pro. If no → Community.

## How `RequirePro` behaves

`libtftest.RequirePro(tb)` checks the resolved edition (set during
`harness.Run` from `LOCALSTACK_AUTH_TOKEN` and the health endpoint). On
Community edition the test calls `tb.Skipf` with a clear message rather than
failing. This means a single test file can contain a mix of Community and
Pro-only assertions and the Community CI run still passes (those
assertions skip).
