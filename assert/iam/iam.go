package iam

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	iamsdk "github.com/aws/aws-sdk-go-v2/service/iam"

	"github.com/donaldgifford/libtftest"
	"github.com/donaldgifford/libtftest/awsx"
)

// RoleExistsContext asserts that the named IAM role exists.
//
// libtftest:requires pro IAM GetRole is not implemented by LocalStack OSS.
func RoleExistsContext(tb testing.TB, ctx context.Context, cfg aws.Config, name string) {
	tb.Helper()
	libtftest.RequirePro(tb)

	client := awsx.NewIAM(cfg)
	_, err := client.GetRole(ctx, &iamsdk.GetRoleInput{
		RoleName: aws.String(name),
	})
	if err != nil {
		tb.Errorf("RoleExists(%q): %v", name, err)
	}
}

// RoleExists is a shim that calls RoleExistsContext with tb.Context().
//
// libtftest:requires pro IAM GetRole is not implemented by LocalStack OSS.
func RoleExists(tb testing.TB, cfg aws.Config, name string) {
	tb.Helper()
	RoleExistsContext(tb, tb.Context(), cfg, name)
}

// RoleHasInlinePolicyContext asserts that the named IAM role carries
// the named inline policy.
//
// libtftest:requires pro IAM GetRolePolicy is not implemented by LocalStack OSS.
func RoleHasInlinePolicyContext(tb testing.TB, ctx context.Context, cfg aws.Config, role, policy string) {
	tb.Helper()
	libtftest.RequirePro(tb)

	client := awsx.NewIAM(cfg)
	_, err := client.GetRolePolicy(ctx, &iamsdk.GetRolePolicyInput{
		RoleName:   aws.String(role),
		PolicyName: aws.String(policy),
	})
	if err != nil {
		tb.Errorf("RoleHasInlinePolicy(%q, %q): %v", role, policy, err)
	}
}

// RoleHasInlinePolicy is a shim that calls RoleHasInlinePolicyContext with tb.Context().
//
// libtftest:requires pro IAM GetRolePolicy is not implemented by LocalStack OSS.
func RoleHasInlinePolicy(tb testing.TB, cfg aws.Config, role, policy string) {
	tb.Helper()
	RoleHasInlinePolicyContext(tb, tb.Context(), cfg, role, policy)
}
