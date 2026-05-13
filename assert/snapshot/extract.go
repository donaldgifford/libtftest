package snapshot

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// ExtractIAMPolicies walks `planned_values.root_module.resources` in a
// `terraform show -json plan.out` payload and returns one entry per
// IAM-policy-bearing resource keyed by the resource address plus a
// suffix that distinguishes the policy slot:
//
//   - `<addr>.assume_role`         — aws_iam_role.assume_role_policy
//   - `<addr>.inline:<name>`       — aws_iam_role_policy (inline policy)
//   - `<addr>.managed:<arn>`       — aws_iam_role_policy_attachment
//   - `<addr>.policy`              — aws_iam_policy.policy
//
// Inline policies and assume-role policies render as full JSON
// documents. Managed-policy attachments and customer-managed-policy
// attachments render as the canonical ARN string — they're effectively
// an enum (AWS owns them; we don't fetch live documents because that
// would make the helper network-dependent and non-deterministic).
//
// The returned map's iteration order is undefined; callers that snapshot
// a single key in isolation should use the returned bytes directly, and
// callers that aggregate must sort the keys explicitly.
func ExtractIAMPolicies(planJSON []byte) (map[string][]byte, error) {
	var plan planFile
	if err := json.Unmarshal(planJSON, &plan); err != nil {
		return nil, fmt.Errorf("unmarshal plan: %w", err)
	}

	out := make(map[string][]byte)
	for _, r := range plan.PlannedValues.RootModule.Resources {
		if err := extractFromResource(r, out); err != nil {
			return nil, err
		}
	}

	return out, nil
}

func extractFromResource(r planResource, out map[string][]byte) error {
	switch r.Type {
	case "aws_iam_role":
		return extractStringPolicy(r, out, "assume_role_policy", ".assume_role")
	case "aws_iam_role_policy":
		name, ok := r.Values["name"].(string)
		if !ok || name == "" {
			return nil
		}
		return extractStringPolicy(r, out, "policy", ".inline:"+name)
	case "aws_iam_role_policy_attachment":
		if arn, ok := r.Values["policy_arn"].(string); ok && arn != "" {
			out[r.Address+".managed:"+arn] = []byte(arn)
		}
		return nil
	case "aws_iam_policy":
		return extractStringPolicy(r, out, "policy", ".policy")
	}
	return nil
}

func extractStringPolicy(r planResource, out map[string][]byte, field, suffix string) error {
	doc, ok := r.Values[field].(string)
	if !ok {
		return nil
	}
	normalized, err := NormalizeJSON([]byte(doc))
	if err != nil {
		return fmt.Errorf("%s%s: %w", r.Address, suffix, err)
	}
	out[r.Address+suffix] = normalized
	return nil
}

// ExtractResourceAttribute returns the JSON bytes at attributePath
// under `planned_values.root_module.resources[?address==resourceAddress].values`.
// attributePath uses dot notation (e.g. `policy`, `tags.Owner`).
//
// Returns an error if the resource address isn't found, the attribute
// path doesn't resolve, or the JSON is malformed.
func ExtractResourceAttribute(planJSON []byte, resourceAddress, attributePath string) ([]byte, error) {
	if resourceAddress == "" {
		return nil, errors.New("resourceAddress is empty")
	}
	if attributePath == "" {
		return nil, errors.New("attributePath is empty")
	}

	var plan planFile
	if err := json.Unmarshal(planJSON, &plan); err != nil {
		return nil, fmt.Errorf("unmarshal plan: %w", err)
	}

	var target map[string]any
	for _, r := range plan.PlannedValues.RootModule.Resources {
		if r.Address == resourceAddress {
			target = r.Values
			break
		}
	}
	if target == nil {
		return nil, fmt.Errorf("resource %q not found in planned_values", resourceAddress)
	}

	var cur any = target
	for segment := range strings.SplitSeq(attributePath, ".") {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("%s: segment %q applied to non-object", resourceAddress, segment)
		}
		next, present := m[segment]
		if !present {
			return nil, fmt.Errorf("%s: attribute %q not present", resourceAddress, attributePath)
		}
		cur = next
	}

	// If the leaf is a JSON-encoded string (the AWS provider's convention
	// for policy docs), return the inner JSON. Otherwise marshal the value.
	if s, ok := cur.(string); ok && looksLikeJSON(s) {
		return NormalizeJSON([]byte(s))
	}
	return marshalSorted(cur)
}

func looksLikeJSON(s string) bool {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return false
	}
	first := trimmed[0]
	return first == '{' || first == '['
}

// planFile is the subset of `terraform show -json` output we consume.
// We do not parse every field — only what the extraction helpers need —
// so the helpers stay resilient to future Terraform output changes.
type planFile struct {
	PlannedValues struct {
		RootModule struct {
			Resources []planResource `json:"resources"`
		} `json:"root_module"`
	} `json:"planned_values"`
}

type planResource struct {
	Address string         `json:"address"`
	Type    string         `json:"type"`
	Values  map[string]any `json:"values"`
}
