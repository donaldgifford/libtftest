package localstack

import "testing"

func TestEdition_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		e    Edition
		want string
	}{
		{name: "auto", e: EditionAuto, want: "auto"},
		{name: "community", e: EditionCommunity, want: "community"},
		{name: "pro", e: EditionPro, want: "pro"},
		{name: "unknown", e: Edition(99), want: "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.e.String(); got != tt.want {
				t.Errorf("Edition(%d).String() = %q, want %q", tt.e, got, tt.want)
			}
		})
	}
}

func TestDetectEdition_ExplicitEdition(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		e    Edition
		want Edition
	}{
		{name: "community stays community", e: EditionCommunity, want: EditionCommunity},
		{name: "pro stays pro", e: EditionPro, want: EditionPro},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := DetectEdition(tt.e); got != tt.want {
				t.Errorf("DetectEdition(%v) = %v, want %v", tt.e, got, tt.want)
			}
		})
	}
}

func TestDetectEdition_AutoWithToken(t *testing.T) {
	t.Setenv("LOCALSTACK_AUTH_TOKEN", "test-token-123")

	got := DetectEdition(EditionAuto)
	if got != EditionPro {
		t.Errorf("DetectEdition(Auto) with token = %v, want Pro", got)
	}
}

func TestDetectEdition_AutoWithoutToken(t *testing.T) {
	t.Setenv("LOCALSTACK_AUTH_TOKEN", "")

	got := DetectEdition(EditionAuto)
	if got != EditionCommunity {
		t.Errorf("DetectEdition(Auto) without token = %v, want Community", got)
	}
}
