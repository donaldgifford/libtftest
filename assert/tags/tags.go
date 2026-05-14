package tags

import (
	"context"
	"fmt"
	"sort"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	rgttsdk "github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi"
	rgtttypes "github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi/types"

	"github.com/donaldgifford/libtftest/awsx"
)

// PropagatesFromRootContext asserts that every key in baseline is
// present (with the expected value) on every ARN listed. Extra tags on
// the resources are allowed — this is a subset check. Failures are
// aggregated across all ARNs and surfaced via a single tb.Errorf call
// so a single test run reports every missing or mismatched tag.
//
// Behavior:
//   - Empty baseline: no-op; returns immediately. Useful for "module
//     has no required tags" tests.
//   - Empty arns: tb.Errorf — caller almost certainly intended to
//     pass at least one resource.
//   - Resource missing from GetResources response: each missing ARN
//     contributes one "not found" failure to the aggregate.
//   - Tag missing on a resource: one "missing tag" failure per
//     (resource, key) pair.
//   - Tag present but wrong value: one "wrong value" failure per
//     (resource, key) pair.
func PropagatesFromRootContext(
	tb testing.TB,
	ctx context.Context,
	cfg aws.Config,
	baseline map[string]string,
	arns ...string,
) {
	tb.Helper()

	if len(baseline) == 0 {
		return
	}
	if len(arns) == 0 {
		tb.Errorf("PropagatesFromRoot: no ARNs supplied")
		return
	}

	client := awsx.NewResourceGroupsTagging(cfg)

	out, err := client.GetResources(ctx, &rgttsdk.GetResourcesInput{
		ResourceARNList: arns,
	})
	if err != nil {
		tb.Errorf("PropagatesFromRoot: GetResources(%v): %v", arns, err)
		return
	}

	if problems := diffTags(baseline, arns, indexByARN(out.ResourceTagMappingList)); len(problems) > 0 {
		tb.Errorf("PropagatesFromRoot: %d tag problem(s):\n  - %s", len(problems), joinWithIndent(problems))
	}
}

// diffTags compares the baseline against the actual tags indexed by ARN
// and returns a sorted slice of human-readable problems. Separated from
// the SDK call so the comparison logic is unit-testable in isolation.
func diffTags(baseline map[string]string, arns []string, got map[string]map[string]string) []string {
	var problems []string
	for _, arn := range arns {
		tags, ok := got[arn]
		if !ok {
			problems = append(problems, fmt.Sprintf("%s: not returned by GetResources", arn))
			continue
		}
		for key, want := range baseline {
			gotVal, present := tags[key]
			switch {
			case !present:
				problems = append(problems, fmt.Sprintf("%s: missing tag %q (want %q)", arn, key, want))
			case gotVal != want:
				problems = append(problems, fmt.Sprintf("%s: tag %q = %q, want %q", arn, key, gotVal, want))
			}
		}
	}
	sort.Strings(problems)
	return problems
}

// PropagatesFromRoot is a shim that calls PropagatesFromRootContext with tb.Context().
func PropagatesFromRoot(
	tb testing.TB,
	cfg aws.Config,
	baseline map[string]string,
	arns ...string,
) {
	tb.Helper()
	PropagatesFromRootContext(tb, tb.Context(), cfg, baseline, arns...)
}

func indexByARN(mappings []rgtttypes.ResourceTagMapping) map[string]map[string]string {
	out := make(map[string]map[string]string, len(mappings))
	for _, m := range mappings {
		if m.ResourceARN == nil {
			continue
		}
		tagMap := make(map[string]string, len(m.Tags))
		for _, t := range m.Tags {
			if t.Key == nil {
				continue
			}
			key := *t.Key
			val := ""
			if t.Value != nil {
				val = *t.Value
			}
			tagMap[key] = val
		}
		out[*m.ResourceARN] = tagMap
	}
	return out
}

func joinWithIndent(lines []string) string {
	out := ""
	for i, line := range lines {
		if i > 0 {
			out += "\n  - "
		}
		out += line
	}
	return out
}
