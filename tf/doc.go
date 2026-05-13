// Package tf handles Terraform workspace management, override
// rendering, and terraform.Options construction for libtftest's
// TestCase fixture.
//
// The package answers three concerns:
//
//   - **Workspace copy** — TestCase copies the module under test
//     into a t.TempDir before plan/apply so concurrent tests don't
//     stomp on each other's .terraform/ state.
//
//   - **Override rendering** — tf writes two JSON overlay files
//     into the copied workspace:
//     _libtftest_override.tf.json (provider config pointed at
//     LocalStack) and _libtftest_backend_override.tf.json (forces
//     backend "local"). Terraform's key-by-key JSON merge means the
//     module's own .tf files stay untouched.
//
//   - **terraform.Options construction** — tf assembles the
//     Terratest options struct from the user's libtftest.Options,
//     the resolved workspace path, and the LocalStack endpoint, so
//     callers don't repeat the boilerplate across tests.
//
// See DESIGN-0001 for the override-file rationale (we pick JSON
// overlay over HCL injection so module authors never have to merge
// libtftest's edits with their own changes).
package tf
