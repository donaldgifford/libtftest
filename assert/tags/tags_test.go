package tags

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	rgtttypes "github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi/types"

	"github.com/donaldgifford/libtftest/internal/testfake"
)

func ptr(s string) *string { return &s }

func mapping(arn string, kv ...string) rgtttypes.ResourceTagMapping {
	if len(kv)%2 != 0 {
		panic("mapping: kv must be even-length")
	}
	tags := make([]rgtttypes.Tag, 0, len(kv)/2)
	for i := 0; i < len(kv); i += 2 {
		tags = append(tags, rgtttypes.Tag{Key: ptr(kv[i]), Value: ptr(kv[i+1])})
	}
	return rgtttypes.ResourceTagMapping{
		ResourceARN: ptr(arn),
		Tags:        tags,
	}
}

func TestIndexByARN(t *testing.T) {
	t.Parallel()

	mappings := []rgtttypes.ResourceTagMapping{
		mapping("arn:a", "owner", "team-x", "env", "prod"),
		mapping("arn:b", "owner", "team-y"),
		{ResourceARN: nil, Tags: []rgtttypes.Tag{{Key: ptr("ignored"), Value: ptr("v")}}},
		{ResourceARN: ptr("arn:c"), Tags: []rgtttypes.Tag{{Key: nil, Value: ptr("skip")}, {Key: ptr("kept"), Value: nil}}},
	}

	got := indexByARN(mappings)

	if len(got) != 3 {
		t.Fatalf("indexByARN returned %d entries, want 3 (nil-ARN skipped)", len(got))
	}
	if got["arn:a"]["owner"] != "team-x" || got["arn:a"]["env"] != "prod" {
		t.Errorf("arn:a tags = %v", got["arn:a"])
	}
	if v, ok := got["arn:c"]["kept"]; !ok || v != "" {
		t.Errorf("arn:c kept tag with nil Value should map to empty string, got (%q, %v)", v, ok)
	}
	if _, ok := got["arn:c"][""]; ok {
		t.Errorf("arn:c nil-Key tag should be skipped, got map %v", got["arn:c"])
	}
}

func TestDiffTags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		baseline map[string]string
		arns     []string
		got      map[string]map[string]string
		want     []string
	}{
		{
			name:     "all present and correct",
			baseline: map[string]string{"owner": "team-x"},
			arns:     []string{"arn:a"},
			got:      map[string]map[string]string{"arn:a": {"owner": "team-x", "extra": "ok"}},
			want:     nil,
		},
		{
			name:     "missing key",
			baseline: map[string]string{"owner": "team-x"},
			arns:     []string{"arn:a"},
			got:      map[string]map[string]string{"arn:a": {"env": "prod"}},
			want:     []string{`arn:a: missing tag "owner" (want "team-x")`},
		},
		{
			name:     "wrong value",
			baseline: map[string]string{"owner": "team-x"},
			arns:     []string{"arn:a"},
			got:      map[string]map[string]string{"arn:a": {"owner": "team-y"}},
			want:     []string{`arn:a: tag "owner" = "team-y", want "team-x"`},
		},
		{
			name:     "resource not returned",
			baseline: map[string]string{"owner": "team-x"},
			arns:     []string{"arn:missing"},
			got:      map[string]map[string]string{},
			want:     []string{"arn:missing: not returned by GetResources"},
		},
		{
			name:     "multi-arn aggregation",
			baseline: map[string]string{"owner": "team-x", "env": "prod"},
			arns:     []string{"arn:a", "arn:b"},
			got: map[string]map[string]string{
				"arn:a": {"owner": "team-x"},
				"arn:b": {"env": "stage"},
			},
			want: []string{
				`arn:a: missing tag "env" (want "prod")`,
				`arn:b: missing tag "owner" (want "team-x")`,
				`arn:b: tag "env" = "stage", want "prod"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := diffTags(tt.baseline, tt.arns, tt.got)
			if len(got) != len(tt.want) {
				t.Fatalf("diffTags() returned %d problems %v, want %d %v",
					len(got), got, len(tt.want), tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("problem[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestPropagatesFromRootContext_EmptyBaseline_NoOp(t *testing.T) {
	t.Parallel()

	tb := testfake.NewFakeTB()
	PropagatesFromRootContext(tb, context.Background(), testCfg(), nil, "arn:a")

	if tb.Errored() {
		t.Error("empty baseline must be a no-op; got Errorf")
	}
}

func TestPropagatesFromRootContext_NoARNs_Errors(t *testing.T) {
	t.Parallel()

	tb := testfake.NewFakeTB()
	PropagatesFromRootContext(tb, context.Background(), testCfg(), map[string]string{"owner": "team-x"})

	if !tb.Errored() {
		t.Error("empty arns must Errorf")
	}
}

func TestPropagatesFromRootContext_PropagatesCancel(t *testing.T) {
	t.Parallel()

	tb := testfake.NewFakeTB()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		PropagatesFromRootContext(tb, ctx, testCfg(), map[string]string{"owner": "team-x"}, "arn:a")
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("PropagatesFromRootContext did not honor cancelled ctx within 2s")
	}

	if !tb.Errored() {
		t.Error("PropagatesFromRootContext with cancelled ctx did not Errorf")
	}
}

func TestJoinWithIndent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   []string
		want string
	}{
		{in: nil, want: ""},
		{in: []string{"a"}, want: "a"},
		{in: []string{"a", "b"}, want: "a\n  - b"},
		{in: []string{"a", "b", "c"}, want: "a\n  - b\n  - c"},
	}

	for _, tt := range tests {
		got := joinWithIndent(tt.in)
		if got != tt.want {
			t.Errorf("joinWithIndent(%v) = %q, want %q", tt.in, got, tt.want)
		}
		if len(tt.in) > 1 && !strings.Contains(got, "\n  - ") {
			t.Errorf("multi-line output should contain the indent prefix")
		}
	}
}

func testCfg() aws.Config {
	return aws.Config{
		Region:       "us-east-1",
		BaseEndpoint: aws.String("http://127.0.0.1:1"),
		Credentials:  credentials.NewStaticCredentialsProvider("test", "test", ""),
	}
}
