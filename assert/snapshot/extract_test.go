package snapshot_test

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"testing"

	"github.com/donaldgifford/libtftest/assert/snapshot"
)

func loadPlan(t *testing.T) []byte {
	t.Helper()
	_, filename, _, _ := runtime.Caller(0) //nolint:dogsled // Only need filename.
	path := filepath.Join(filepath.Dir(filename), "testdata", "plan-iam-kms.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read plan fixture: %v", err)
	}
	return data
}

func TestExtractIAMPolicies_Shape(t *testing.T) {
	t.Parallel()

	planJSON := loadPlan(t)

	got, err := snapshot.ExtractIAMPolicies(planJSON)
	if err != nil {
		t.Fatalf("ExtractIAMPolicies: %v", err)
	}

	wantKeys := []string{
		"aws_iam_policy.deploy.policy",
		"aws_iam_role.eks_node.assume_role",
		"aws_iam_role_policy.eks_node_inline.inline:eks-node-extra",
		"aws_iam_role_policy_attachment.eks_node_worker.managed:arn:aws:iam::aws:policy/AmazonEKSWorkerNodePolicy",
	}

	gotKeys := make([]string, 0, len(got))
	for k := range got {
		gotKeys = append(gotKeys, k)
	}
	sort.Strings(gotKeys)
	sort.Strings(wantKeys)

	if len(gotKeys) != len(wantKeys) {
		t.Fatalf("ExtractIAMPolicies returned %d keys %v, want %d %v",
			len(gotKeys), gotKeys, len(wantKeys), wantKeys)
	}
	for i := range gotKeys {
		if gotKeys[i] != wantKeys[i] {
			t.Errorf("key[%d] = %q, want %q", i, gotKeys[i], wantKeys[i])
		}
	}
}

func TestExtractIAMPolicies_ManagedRendersARNNotDocument(t *testing.T) {
	t.Parallel()

	got, err := snapshot.ExtractIAMPolicies(loadPlan(t))
	if err != nil {
		t.Fatal(err)
	}

	const arn = "arn:aws:iam::aws:policy/AmazonEKSWorkerNodePolicy"
	const key = "aws_iam_role_policy_attachment.eks_node_worker.managed:" + arn

	payload, ok := got[key]
	if !ok {
		t.Fatalf("expected managed key %q in result; got keys %v", key, got)
	}
	if string(payload) != arn {
		t.Errorf("managed policy value = %q, want %q (ARN string, NOT the live document)",
			string(payload), arn)
	}
}

func TestExtractIAMPolicies_DocumentsAreNormalized(t *testing.T) {
	t.Parallel()

	got, err := snapshot.ExtractIAMPolicies(loadPlan(t))
	if err != nil {
		t.Fatal(err)
	}

	const key = "aws_iam_role.eks_node.assume_role"
	const want = `{"Statement":[{"Action":"sts:AssumeRole","Effect":"Allow","Principal":{"Service":"ec2.amazonaws.com"}}],"Version":"2012-10-17"}`

	if string(got[key]) != want {
		t.Errorf("assume_role policy = %q\nwant %q (normalized: keys sorted)",
			string(got[key]), want)
	}
}

func TestExtractIAMPolicies_DeterministicAcrossRuns(t *testing.T) {
	t.Parallel()

	planJSON := loadPlan(t)

	a, err := snapshot.ExtractIAMPolicies(planJSON)
	if err != nil {
		t.Fatal(err)
	}
	b, err := snapshot.ExtractIAMPolicies(planJSON)
	if err != nil {
		t.Fatal(err)
	}

	if len(a) != len(b) {
		t.Fatalf("two runs returned different key counts: %d vs %d", len(a), len(b))
	}
	for k, va := range a {
		vb, ok := b[k]
		if !ok {
			t.Errorf("key %q present in run 1 but missing in run 2", k)
			continue
		}
		if !bytes.Equal(va, vb) {
			t.Errorf("key %q value differs across runs:\n  %q\n  %q",
				k, string(va), string(vb))
		}
	}
}

func TestExtractIAMPolicies_InvalidJSON(t *testing.T) {
	t.Parallel()

	_, err := snapshot.ExtractIAMPolicies([]byte("not-json"))
	if err == nil {
		t.Error("ExtractIAMPolicies on invalid JSON should error")
	}
}

func TestExtractResourceAttribute_KMSPolicy(t *testing.T) {
	t.Parallel()

	got, err := snapshot.ExtractResourceAttribute(loadPlan(t), "aws_kms_key.main", "policy")
	if err != nil {
		t.Fatalf("ExtractResourceAttribute: %v", err)
	}

	const want = `{"Statement":[{"Action":"kms:*","Effect":"Allow","Principal":{"AWS":"arn:aws:iam::123456789012:root"},"Resource":"*"}],"Version":"2012-10-17"}`
	if string(got) != want {
		t.Errorf("KMS policy:\n  got:  %s\n  want: %s", string(got), want)
	}
}

func TestExtractResourceAttribute_S3TagMap(t *testing.T) {
	t.Parallel()

	got, err := snapshot.ExtractResourceAttribute(loadPlan(t), "aws_s3_bucket.assets", "tags")
	if err != nil {
		t.Fatalf("ExtractResourceAttribute: %v", err)
	}

	const want = `{"Env":"prod","Owner":"platform"}`
	if string(got) != want {
		t.Errorf("tags map:\n  got:  %s\n  want: %s", string(got), want)
	}
}

func TestExtractResourceAttribute_MissingResource(t *testing.T) {
	t.Parallel()

	_, err := snapshot.ExtractResourceAttribute(loadPlan(t), "aws_iam_role.nope", "policy")
	if err == nil {
		t.Error("missing resource should error")
	}
}

func TestExtractResourceAttribute_MissingAttribute(t *testing.T) {
	t.Parallel()

	_, err := snapshot.ExtractResourceAttribute(loadPlan(t), "aws_s3_bucket.assets", "nope")
	if err == nil {
		t.Error("missing attribute should error")
	}
}

func TestExtractResourceAttribute_EmptyArgs(t *testing.T) {
	t.Parallel()

	plan := loadPlan(t)

	if _, err := snapshot.ExtractResourceAttribute(plan, "", "policy"); err == nil {
		t.Error("empty resourceAddress should error")
	}
	if _, err := snapshot.ExtractResourceAttribute(plan, "aws_kms_key.main", ""); err == nil {
		t.Error("empty attributePath should error")
	}
}
