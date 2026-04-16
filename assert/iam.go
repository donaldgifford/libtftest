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

// RoleExists asserts that the named IAM role exists. Pro-only: calls RequirePro.
func (iamAsserts) RoleExists(tb testing.TB, cfg aws.Config, name string) {
	tb.Helper()
	libtftest.RequirePro(tb)

	client := awsx.NewIAM(cfg)
	_, err := client.GetRole(context.Background(), &iam.GetRoleInput{
		RoleName: aws.String(name),
	})
	if err != nil {
		tb.Errorf("RoleExists(%q): %v", name, err)
	}
}

// RoleHasInlinePolicy asserts that the role has the named inline policy. Pro-only.
func (iamAsserts) RoleHasInlinePolicy(tb testing.TB, cfg aws.Config, role, policy string) {
	tb.Helper()
	libtftest.RequirePro(tb)

	client := awsx.NewIAM(cfg)
	_, err := client.GetRolePolicy(context.Background(), &iam.GetRolePolicyInput{
		RoleName:   aws.String(role),
		PolicyName: aws.String(policy),
	})
	if err != nil {
		tb.Errorf("RoleHasInlinePolicy(%q, %q): %v", role, policy, err)
	}
}
