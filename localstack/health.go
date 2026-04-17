package localstack

import (
	"encoding/json"
	"fmt"
	"io"
)

// HealthResponse represents the JSON response from /_localstack/health.
type HealthResponse struct {
	Edition  string            `json:"edition"`
	Version  string            `json:"version"`
	Services map[string]string `json:"services"`
}

// AllServicesReady returns true if no service is in state "initializing" or
// "error". The signature matches testcontainers' wait.HTTPStrategy
// ResponseMatcher: func(body io.Reader) bool.
func AllServicesReady(body io.Reader) bool {
	data, err := io.ReadAll(body)
	if err != nil {
		return false
	}

	var hr HealthResponse
	if err := json.Unmarshal(data, &hr); err != nil {
		return false
	}

	for _, status := range hr.Services {
		if status == "initializing" || status == "error" {
			return false
		}
	}

	return true
}

// ParseHealth parses a raw health response body into a HealthResponse.
func ParseHealth(body []byte) (*HealthResponse, error) {
	var hr HealthResponse
	if err := json.Unmarshal(body, &hr); err != nil {
		return nil, fmt.Errorf("parse health response: %w", err)
	}

	return &hr, nil
}

// DetectEditionFromHealth returns the Edition based on the health response's
// edition field.
func DetectEditionFromHealth(body []byte) Edition {
	hr, err := ParseHealth(body)
	if err != nil {
		return EditionCommunity
	}

	switch hr.Edition {
	case editionProLabel:
		return EditionPro
	default:
		return EditionCommunity
	}
}
