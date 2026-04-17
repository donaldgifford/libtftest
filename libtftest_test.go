package libtftest

import (
	"testing"
)

func TestSetVar(t *testing.T) {
	t.Parallel()

	tc := &TestCase{
		vars: map[string]any{"existing": "value"},
	}

	tc.SetVar("new_key", "new_value")

	if tc.vars["new_key"] != "new_value" {
		t.Errorf("SetVar() did not set key: got %v", tc.vars["new_key"])
	}

	// Overwrite existing key.
	tc.SetVar("existing", "updated")
	if tc.vars["existing"] != "updated" {
		t.Errorf("SetVar() did not overwrite: got %v", tc.vars["existing"])
	}
}

func TestParsePlanChanges(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		json    string
		want    PlanChanges
		wantErr bool
	}{
		{
			name: "create resources",
			json: `{
				"format_version": "1.2",
				"resource_changes": [
					{"change": {"actions": ["create"]}},
					{"change": {"actions": ["create"]}}
				]
			}`,
			want: PlanChanges{Add: 2},
		},
		{
			name: "update and delete",
			json: `{
				"format_version": "1.2",
				"resource_changes": [
					{"change": {"actions": ["update"]}},
					{"change": {"actions": ["delete"]}}
				]
			}`,
			want: PlanChanges{Change: 1, Destroy: 1},
		},
		{
			name: "replace counts as destroy + add",
			json: `{
				"format_version": "1.2",
				"resource_changes": [
					{"change": {"actions": ["delete", "create"]}}
				]
			}`,
			want: PlanChanges{Add: 1, Destroy: 1},
		},
		{
			name: "no-op changes",
			json: `{
				"format_version": "1.2",
				"resource_changes": [
					{"change": {"actions": ["no-op"]}}
				]
			}`,
			want: PlanChanges{},
		},
		{
			name: "empty plan",
			json: `{
				"format_version": "1.2",
				"resource_changes": []
			}`,
			want: PlanChanges{},
		},
		{
			name:    "invalid json",
			json:    `not json`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parsePlanChanges([]byte(tt.json))
			if tt.wantErr {
				if err == nil {
					t.Error("parsePlanChanges() = nil error, want error")
				}
				return
			}

			if err != nil {
				t.Fatalf("parsePlanChanges() error = %v", err)
			}

			if got != tt.want {
				t.Errorf("parsePlanChanges() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestPrefix(t *testing.T) {
	t.Parallel()

	tc := &TestCase{prefix: "ltt-abc123"}

	if got := tc.Prefix(); got != "ltt-abc123" {
		t.Errorf("Prefix() = %q, want %q", got, "ltt-abc123")
	}
}
