// Package snapshot provides JSON snapshot testing for Terraform plans
// and other deterministic JSON payloads, plus a small extraction
// toolkit for the IAM-heavy use cases that motivate it.
//
// Three building blocks:
//
//   - [JSONStrict] — byte-for-byte comparison against a snapshot file.
//     Use when key order is semantically meaningful.
//   - [JSONStructural] — normalizes both sides (recursively sorts keys
//     and strips insignificant whitespace) before comparing. Use for
//     IAM policies, Terraform plan JSON, and anywhere key order is
//     arbitrary.
//   - [ExtractIAMPolicies] / [ExtractResourceAttribute] — pull JSON
//     payloads out of `terraform show -json plan.out` output ready to
//     feed into JSONStructural.
//
// # The UPDATE_SNAPSHOTS=1 protocol
//
// When LIBTFTEST_UPDATE_SNAPSHOTS=1 is set in the environment, missing
// or mismatched snapshots are overwritten with the actual payload and
// the test passes (with a tb.Logf record). This matches the Jest
// snapshot workflow and the `go-cmp` "regenerate goldens" pattern.
// Always commit the regenerated snapshots — they ARE the test.
//
// # Determinism guarantees
//
// All helpers are pure functions of their byte inputs. None of them
// make network calls. ExtractIAMPolicies in particular renders AWS
// managed policy attachments as the canonical ARN string rather than
// fetching the live document — the ARN is effectively an enum AWS
// owns, and fetching would make the helper non-deterministic and
// network-dependent. The same applies to customer-managed policies
// attached by ARN: the snapshot captures the attachment, not the
// policy document, which is a separate snapshot if the module owns
// the document.
package snapshot
