package localstack

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestAllServicesReady(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		body       string
		want       bool
	}{
		{
			name:       "all running",
			statusCode: http.StatusOK,
			body:       `{"edition":"community","version":"3.5.0","services":{"s3":"running","sqs":"running"}}`,
			want:       true,
		},
		{
			name:       "one initializing",
			statusCode: http.StatusOK,
			body:       `{"edition":"community","version":"3.5.0","services":{"s3":"running","sqs":"initializing"}}`,
			want:       false,
		},
		{
			name:       "one error",
			statusCode: http.StatusOK,
			body:       `{"edition":"community","version":"3.5.0","services":{"s3":"error"}}`,
			want:       false,
		},
		{
			name:       "empty services",
			statusCode: http.StatusOK,
			body:       `{"edition":"community","version":"3.5.0","services":{}}`,
			want:       true,
		},
		{
			name:       "non-200 status",
			statusCode: http.StatusServiceUnavailable,
			body:       `{}`,
			want:       false,
		},
		{
			name:       "invalid json",
			statusCode: http.StatusOK,
			body:       `not json`,
			want:       false,
		},
		{
			name:       "available status",
			statusCode: http.StatusOK,
			body:       `{"edition":"pro","version":"3.5.0","services":{"s3":"available","iam":"available"}}`,
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resp := &http.Response{
				StatusCode: tt.statusCode,
				Body:       io.NopCloser(strings.NewReader(tt.body)),
			}

			got := AllServicesReady(resp)
			if got != tt.want {
				t.Errorf("AllServicesReady() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseHealth(t *testing.T) {
	t.Parallel()

	body := []byte(`{
		"edition": "pro",
		"version": "3.5.0",
		"services": {
			"s3": "running",
			"sqs": "available",
			"iam": "running"
		}
	}`)

	hr, err := ParseHealth(body)
	if err != nil {
		t.Fatalf("ParseHealth() error = %v", err)
	}

	if hr.Edition != "pro" {
		t.Errorf("ParseHealth().Edition = %q, want %q", hr.Edition, "pro")
	}

	if hr.Version != "3.5.0" {
		t.Errorf("ParseHealth().Version = %q, want %q", hr.Version, "3.5.0")
	}

	if len(hr.Services) != 3 {
		t.Errorf("ParseHealth().Services has %d entries, want 3", len(hr.Services))
	}

	if hr.Services["s3"] != "running" {
		t.Errorf("ParseHealth().Services[s3] = %q, want %q", hr.Services["s3"], "running")
	}
}

func TestParseHealth_InvalidJSON(t *testing.T) {
	t.Parallel()

	_, err := ParseHealth([]byte("not json"))
	if err == nil {
		t.Error("ParseHealth(invalid) = nil error, want error")
	}
}

func TestDetectEditionFromHealth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
		want Edition
	}{
		{
			name: "pro edition",
			body: `{"edition":"pro","version":"3.5.0","services":{}}`,
			want: EditionPro,
		},
		{
			name: "community edition",
			body: `{"edition":"community","version":"3.5.0","services":{}}`,
			want: EditionCommunity,
		},
		{
			name: "empty edition field",
			body: `{"edition":"","version":"3.5.0","services":{}}`,
			want: EditionCommunity,
		},
		{
			name: "invalid json defaults to community",
			body: `not json`,
			want: EditionCommunity,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := DetectEditionFromHealth([]byte(tt.body))
			if got != tt.want {
				t.Errorf("DetectEditionFromHealth() = %v, want %v", got, tt.want)
			}
		})
	}
}
