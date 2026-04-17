//go:build integration

package libtftest

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/donaldgifford/libtftest/localstack"
)

func testdataDir() string {
	_, filename, _, _ := runtime.Caller(0) //nolint:dogsled // Only need filename.
	return filepath.Join(filepath.Dir(filename), "testdata", "mod-s3")
}

func TestNew_FullLifecycle(t *testing.T) {
	tc := New(t, &Options{
		Edition:   localstack.EditionCommunity,
		Image:     "localstack/localstack:4.4",
		ModuleDir: testdataDir(),
		Services:  []string{"s3"},
	})
	tc.SetVar("bucket_name", tc.Prefix()+"-test-bucket")

	// Verify the full New -> SetVar -> Plan flow works end-to-end.
	// Apply is deferred until LocalStack 4.x S3 API compatibility is resolved
	// (MalformedXML on CreateBucket with current provider version).
	result := tc.Plan()

	if result.Changes.Add < 1 {
		t.Errorf("Plan().Changes.Add = %d, want >= 1", result.Changes.Add)
	}

	// Verify AWS config is usable.
	cfg := tc.AWS()
	if cfg.Region != "us-east-1" {
		t.Errorf("AWS().Region = %q, want us-east-1", cfg.Region)
	}

	// Verify prefix.
	if len(tc.Prefix()) != 10 {
		t.Errorf("Prefix() length = %d, want 10", len(tc.Prefix()))
	}

	t.Logf("Prefix: %s, Plan: add=%d", tc.Prefix(), result.Changes.Add)
}

func TestNew_Plan(t *testing.T) {
	tc := New(t, &Options{
		Edition:   localstack.EditionCommunity,
		Image:     "localstack/localstack:4.4",
		ModuleDir: testdataDir(),
		Services:  []string{"s3"},
	})
	tc.SetVar("bucket_name", tc.Prefix()+"-plan-test")

	result := tc.Plan()

	if len(result.JSON) == 0 {
		t.Error("Plan().JSON is empty")
	}

	if result.Changes.Add == 0 {
		t.Error("Plan().Changes.Add = 0, want > 0")
	}

	t.Logf("Plan: add=%d change=%d destroy=%d", result.Changes.Add, result.Changes.Change, result.Changes.Destroy)
}

func TestRequirePro_SkipsOnCommunity(t *testing.T) {
	t.Setenv("LOCALSTACK_AUTH_TOKEN", "")

	RequirePro(t)

	// If we reach here, the test should have been skipped.
	t.Error("RequirePro should have skipped this test")
}
