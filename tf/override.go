package tf

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	providerOverrideFile = "_libtftest_override.tf.json"
	backendOverrideFile  = "_libtftest_backend_override.tf.json"
)

// services is the initial LocalStack service catalog. Each entry becomes an
// endpoint in the provider override. This list covers the high-traffic services
// from DESIGN-0001; additional services can be added as needed.
var services = []string{
	"s3",
	"dynamodb",
	"iam",
	"sts",
	"ssm",
	"secretsmanager",
	"sqs",
	"sns",
	"lambda",
	"kms",
	"cloudwatch",
	"logs",
	"events",
	"kinesis",
	"firehose",
	"ec2",
	"route53",
	"acm",
	"cloudformation",
	"stepfunctions",
	"cognitoidp",
}

// providerOverride represents the _libtftest_override.tf.json structure.
type providerOverride struct {
	Provider map[string]any `json:"provider"`
}

// backendOverride represents the _libtftest_backend_override.tf.json structure.
type backendOverride struct {
	Terraform map[string]any `json:"terraform"`
}

// RenderProviderOverride generates the provider override JSON that routes
// all AWS services to the given edge URL.
func RenderProviderOverride(edgeURL string) ([]byte, error) {
	endpoints := make(map[string]string, len(services))
	for _, svc := range services {
		endpoints[svc] = edgeURL
	}

	override := providerOverride{
		Provider: map[string]any{
			"aws": map[string]any{
				"region":                      "us-east-1",
				"access_key":                  "test",
				"secret_key":                  "test",
				"skip_credentials_validation": true,
				"skip_metadata_api_check":     true,
				"skip_requesting_account_id":  true,
				"s3_use_path_style":           true,
				"endpoints":                   endpoints,
			},
		},
	}

	data, err := json.MarshalIndent(override, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal provider override: %w", err)
	}

	return data, nil
}

// RenderBackendOverride generates the backend override JSON that forces
// terraform to use a local backend.
func RenderBackendOverride() []byte {
	override := backendOverride{
		Terraform: map[string]any{
			"backend": map[string]any{
				"local": map[string]any{},
			},
		},
	}

	data, err := json.MarshalIndent(override, "", "  ")
	if err != nil {
		// This should never happen with a static structure, but return
		// empty bytes rather than panicking.
		return nil
	}
	return data
}

// WriteOverrides writes both the provider and backend override files into dir.
func WriteOverrides(dir, edgeURL string) error {
	provider, err := RenderProviderOverride(edgeURL)
	if err != nil {
		return fmt.Errorf("render provider override: %w", err)
	}

	providerPath := filepath.Join(dir, providerOverrideFile)
	if err := os.WriteFile(providerPath, provider, 0o640); err != nil {
		return fmt.Errorf("write provider override: %w", err)
	}

	backendPath := filepath.Join(dir, backendOverrideFile)
	if err := os.WriteFile(backendPath, RenderBackendOverride(), 0o640); err != nil {
		return fmt.Errorf("write backend override: %w", err)
	}

	return nil
}
