package assert

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"

	"github.com/donaldgifford/libtftest"
	"github.com/donaldgifford/libtftest/awsx"
)

type iamAsserts struct{}

// RoleExistsContext is the ctx-aware variant of RoleExists. Pro-only: calls RequirePro.
func (iamAsserts) RoleExistsContext(tb testing.TB, ctx context.Context, cfg aws.Config, name string) {
	tb.Helper()
	libtftest.RequirePro(tb)

	client := awsx.NewIAM(cfg)
	_, err := client.GetRole(ctx, &iam.GetRoleInput{
		RoleName: aws.String(name),
	})
	if err != nil {
		tb.Errorf("RoleExists(%q): %v", name, err)
	}
}

// RoleExists is a shim that calls RoleExistsContext with tb.Context().
func (i iamAsserts) RoleExists(tb testing.TB, cfg aws.Config, name string) {
	tb.Helper()
	i.RoleExistsContext(tb, tb.Context(), cfg, name)
}

// RoleHasInlinePolicyContext is the ctx-aware variant of RoleHasInlinePolicy. Pro-only.
func (iamAsserts) RoleHasInlinePolicyContext(tb testing.TB, ctx context.Context, cfg aws.Config, role, policy string) {
	tb.Helper()
	libtftest.RequirePro(tb)

	client := awsx.NewIAM(cfg)
	_, err := client.GetRolePolicy(ctx, &iam.GetRolePolicyInput{
		RoleName:   aws.String(role),
		PolicyName: aws.String(policy),
	})
	if err != nil {
		tb.Errorf("RoleHasInlinePolicy(%q, %q): %v", role, policy, err)
	}
}

// RoleHasInlinePolicy is a shim that calls RoleHasInlinePolicyContext with tb.Context().
func (i iamAsserts) RoleHasInlinePolicy(tb testing.TB, cfg aws.Config, role, policy string) {
	tb.Helper()
	i.RoleHasInlinePolicyContext(tb, tb.Context(), cfg, role, policy)
}
