// Package libtftest wraps Terratest with opinionated, LocalStack-aware defaults
// for Terraform module integration testing.
package libtftest

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/gruntwork-io/terratest/modules/terraform"
	tfjson "github.com/hashicorp/terraform-json"

	"github.com/donaldgifford/libtftest/harness"
	"github.com/donaldgifford/libtftest/internal/logx"
	"github.com/donaldgifford/libtftest/internal/naming"
	"github.com/donaldgifford/libtftest/localstack"
	"github.com/donaldgifford/libtftest/tf"
)

// TestCase is the primary handle returned from New. It owns a LocalStack
// container (or a reference to a shared one), a scratch workspace, and the
// AWS SDK config used for seeding and assertions.
type TestCase struct {
	tb          testing.TB
	stack       *localstack.Container
	work        *tf.Workspace
	awsCfg      aws.Config
	prefix      string
	vars        map[string]any
	opts        Options
	ownStack    bool   // true if this TestCase started the container.
	artifactDir string // resolved artifact directory.
}

// Options configure a TestCase.
type Options struct {
	Edition          localstack.Edition
	Services         []string
	Image            string
	ModuleDir        string
	Vars             map[string]any
	Reuse            *localstack.Container
	PersistOnFailure bool
	InitHooks        []localstack.InitHook
	AutoPrefixVars   bool
	EdgeURLOverride  string
}

// PlanResult holds the output of a terraform plan.
type PlanResult struct {
	JSON     []byte      // Raw `terraform show -json` output.
	FilePath string      // Path to the binary plan file.
	Changes  PlanChanges // Parsed summary of resource changes.
}

// PlanChanges summarizes the resource-level diff from a plan.
type PlanChanges struct {
	Add     int // Resources to create.
	Change  int // Resources to update in-place.
	Destroy int // Resources to destroy.
}

// New creates a TestCase. It starts LocalStack (or attaches to a shared one),
// copies the module into a scratch workspace, writes the provider override,
// and registers cleanup with t.Cleanup. It calls t.Fatal on any setup error.
func New(tb testing.TB, opts *Options) *TestCase {
	tb.Helper()

	tc := &TestCase{
		tb:   tb,
		opts: *opts,
		vars: make(map[string]any),
	}

	// Merge initial vars.
	for k, v := range opts.Vars {
		tc.vars[k] = v
	}

	// Generate parallel-safe prefix.
	tc.prefix = naming.Prefix(tb)

	// Auto-inject prefix into name_prefix var if opted in.
	if opts.AutoPrefixVars {
		tc.vars["name_prefix"] = tc.prefix
	}

	// Resolve container: reuse, shared (harness), or start new.
	ctx := context.Background()
	tc.resolveContainer(ctx)

	// Determine edge URL.
	edgeURL := tc.stack.EdgeURL
	if opts.EdgeURLOverride != "" {
		edgeURL = opts.EdgeURLOverride
	}

	// Create scratch workspace and write overrides.
	tc.work = tf.NewWorkspace(tb, opts.ModuleDir)
	if err := tf.WriteOverrides(tc.work.Dir, edgeURL); err != nil {
		tb.Fatalf("write overrides: %v", err)
	}

	// Build AWS config pointed at LocalStack.
	awsCfg, err := config.LoadDefaultConfig(ctx,
		config.WithBaseEndpoint(edgeURL),
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider("test", "test", ""),
		),
	)
	if err != nil {
		tb.Fatalf("build aws config: %v", err)
	}
	tc.awsCfg = awsCfg

	// Resolve artifact directory.
	tc.artifactDir = logx.ResolveArtifactDir(tb, tc.work.Dir)

	// Register cleanup in LIFO order:
	// 1. Container stop (registered first, runs last).
	// 2. Terraform destroy (registered second, runs second).
	// 3. Log flush + artifact dump (registered last, runs first).
	tc.registerCleanup()

	return tc
}

// SetVar sets or overrides a single Terraform variable.
func (tc *TestCase) SetVar(key string, val any) {
	tc.vars[key] = val
}

// Apply runs terraform init + terraform apply and returns the terraform.Options
// so callers can chain additional operations.
func (tc *TestCase) Apply() *terraform.Options {
	tc.tb.Helper()

	tfOpts := tf.BuildOptions(tc.tb, tc.work.Dir, tc.vars)
	//nolint:staticcheck // SA1019: terratest 1.0 *Context variants tracked for migration in INV-0001.
	terraform.InitAndApply(tc.tb, tfOpts)

	return tfOpts
}

// ApplyE is the error-returning variant for negative tests.
func (tc *TestCase) ApplyE() (*terraform.Options, error) {
	tc.tb.Helper()

	tfOpts := tf.BuildOptions(tc.tb, tc.work.Dir, tc.vars)
	//nolint:staticcheck // SA1019: terratest 1.0 *Context variants tracked for migration in INV-0001.
	_, err := terraform.InitAndApplyE(tc.tb, tfOpts)

	return tfOpts, err
}

// Plan runs terraform init + terraform plan -out and returns a PlanResult.
func (tc *TestCase) Plan() *PlanResult {
	tc.tb.Helper()

	result, err := tc.PlanE()
	if err != nil {
		tc.tb.Fatalf("plan: %v", err)
	}

	return result
}

// PlanE is the error-returning variant.
func (tc *TestCase) PlanE() (*PlanResult, error) {
	tc.tb.Helper()

	tfOpts := tf.BuildPlanOptions(tc.tb, tc.work.Dir, tc.vars)

	//nolint:staticcheck // SA1019: terratest 1.0 *Context variants tracked for migration in INV-0001.
	_, err := terraform.InitAndPlanE(tc.tb, tfOpts)
	if err != nil {
		return nil, fmt.Errorf("init and plan: %w", err)
	}

	//nolint:staticcheck // SA1019: terratest 1.0 *Context variants tracked for migration in INV-0001.
	planJSON, err := terraform.ShowE(tc.tb, tfOpts)
	if err != nil {
		return nil, fmt.Errorf("terraform show: %w", err)
	}

	changes, err := parsePlanChanges([]byte(planJSON))
	if err != nil {
		return nil, fmt.Errorf("parse plan changes: %w", err)
	}

	return &PlanResult{
		JSON:     []byte(planJSON),
		FilePath: tfOpts.PlanFilePath,
		Changes:  changes,
	}, nil
}

// Output reads a single Terraform output value.
func (tc *TestCase) Output(name string) string {
	tc.tb.Helper()

	tfOpts := tf.BuildOptions(tc.tb, tc.work.Dir, tc.vars)

	//nolint:staticcheck // SA1019: terratest 1.0 *Context variants tracked for migration in INV-0001.
	return terraform.Output(tc.tb, tfOpts, name)
}

// AWS returns a cached aws.Config pointed at the LocalStack container.
func (tc *TestCase) AWS() aws.Config {
	return tc.awsCfg
}

// Prefix returns the unique string for this test case.
func (tc *TestCase) Prefix() string {
	return tc.prefix
}

// resolveContainer finds or starts a LocalStack container.
func (tc *TestCase) resolveContainer(ctx context.Context) {
	tc.tb.Helper()

	// Priority: explicit Reuse > harness.Current() > start new.
	if tc.opts.Reuse != nil {
		tc.stack = tc.opts.Reuse
		tc.ownStack = false
		return
	}

	// Check for shared container from harness.Run.
	if ctr := harness.Current(); ctr != nil {
		tc.stack = ctr
		tc.ownStack = false
		return
	}

	// Start a new container.
	cfg := &localstack.Config{
		Edition:   tc.opts.Edition,
		Image:     tc.opts.Image,
		Services:  tc.opts.Services,
		InitHooks: tc.opts.InitHooks,
	}

	ctr, err := localstack.Start(ctx, cfg)
	if err != nil {
		tc.tb.Fatalf("start localstack: %v", err)
	}

	tc.stack = ctr
	tc.ownStack = true
}

// registerCleanup registers t.Cleanup callbacks in the correct LIFO order.
func (tc *TestCase) registerCleanup() {
	// 1. Container stop (registered first, runs last in LIFO).
	if tc.ownStack {
		tc.tb.Cleanup(func() {
			if tc.opts.PersistOnFailure && tc.tb.Failed() {
				tc.tb.Logf("PersistOnFailure: container %s kept alive at %s", tc.stack.ID, tc.stack.EdgeURL)
				return
			}

			if err := tc.stack.Stop(context.Background()); err != nil {
				tc.tb.Errorf("stop container: %v", err)
			}
		})
	}

	// 2. Terraform destroy (registered second, runs second in LIFO).
	tc.tb.Cleanup(func() {
		if tc.opts.PersistOnFailure && tc.tb.Failed() {
			return
		}

		tfOpts := tf.BuildOptions(tc.tb, tc.work.Dir, tc.vars)
		//nolint:staticcheck // SA1019: terratest 1.0 *Context variants tracked for migration in INV-0001.
		if _, err := terraform.DestroyE(tc.tb, tfOpts); err != nil {
			tc.tb.Errorf("terraform destroy: %v", err)
		}
	})

	// 3. Artifact dump (registered last, runs first in LIFO).
	tc.tb.Cleanup(func() {
		if !tc.tb.Failed() {
			return
		}

		tc.dumpArtifacts()
	})
}

// dumpArtifacts saves debug info on test failure.
func (tc *TestCase) dumpArtifacts() {
	dumpIfExists := func(src, name string) {
		data, err := os.ReadFile(src)
		if err != nil {
			return // File doesn't exist or isn't readable.
		}
		logx.DumpArtifact(tc.tb, tc.artifactDir, name, data)
	}

	dumpIfExists(tc.work.Dir+"/_libtftest_override.tf.json", "provider-override.json")
	dumpIfExists(tc.work.Dir+"/_libtftest_backend_override.tf.json", "backend-override.json")
	dumpIfExists(tc.work.Dir+"/libtftest.plan", "plan.bin")
}

// parsePlanChanges extracts resource change counts from terraform show -json output.
func parsePlanChanges(planJSON []byte) (PlanChanges, error) {
	var plan tfjson.Plan
	if err := json.Unmarshal(planJSON, &plan); err != nil {
		return PlanChanges{}, fmt.Errorf("unmarshal plan: %w", err)
	}

	var changes PlanChanges
	for _, rc := range plan.ResourceChanges {
		if rc.Change == nil {
			continue
		}

		actions := rc.Change.Actions
		switch {
		case actions.Create():
			changes.Add++
		case actions.Update():
			changes.Change++
		case actions.Delete():
			changes.Destroy++
		case actions.Replace():
			changes.Destroy++
			changes.Add++
		}
	}

	return changes, nil
}

// RequirePro skips the test when the running container is Community edition.
func RequirePro(tb testing.TB) {
	tb.Helper()
	// Implementation depends on the running container's health endpoint.
	// Deferred to integration with TestCase — for now, check env var.
	if os.Getenv("LOCALSTACK_AUTH_TOKEN") == "" {
		tb.Skip("skipping: requires LocalStack Pro (no LOCALSTACK_AUTH_TOKEN set)")
	}
}

// RequireServices skips the test when any of the named services is not
// available in the running container's edition.
func RequireServices(tb testing.TB, _ ...string) {
	tb.Helper()
	// Full implementation requires querying the health endpoint.
	// For now, this is a no-op — all services are assumed available.
}
