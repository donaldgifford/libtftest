package localstack

import "os"

const editionProLabel = "pro"

// Edition represents a LocalStack product edition.
type Edition int

const (
	// EditionAuto detects the edition from the LOCALSTACK_AUTH_TOKEN env var.
	// If the token is set, assumes Pro; otherwise Community.
	EditionAuto Edition = iota
	// EditionCommunity is the free, open-source LocalStack edition.
	EditionCommunity
	// EditionPro is the commercial LocalStack edition with additional services.
	EditionPro
)

// String returns the human-readable edition name.
func (e Edition) String() string {
	switch e {
	case EditionCommunity:
		return "community"
	case EditionPro:
		return editionProLabel
	case EditionAuto:
		return "auto"
	default:
		return "unknown"
	}
}

// DetectEdition resolves EditionAuto to a concrete edition based on the
// LOCALSTACK_AUTH_TOKEN environment variable.
func DetectEdition(e Edition) Edition {
	if e != EditionAuto {
		return e
	}

	if os.Getenv("LOCALSTACK_AUTH_TOKEN") != "" {
		return EditionPro
	}

	return EditionCommunity
}
