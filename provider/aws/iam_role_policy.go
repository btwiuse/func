//nolint: lll
//go:generate go run ../../tools/structdoc/main.go --file $GOFILE --struct IAMRolePolicy --template ../../tools/structdoc/template.txt --data type=aws_iam_role_policy --output ../../docs/resources/aws/iam_role_policy.md

package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/func/func/resource"
	"github.com/pkg/errors"
)

// IAMRolePolicy is an inline role policy, attached to a role.
type IAMRolePolicy struct {
	// Inputs

	// The policy document.
	PolicyDocument string `input:"policy_document"`

	// The name of the policy document.
	PolicyName string `input:"policy_name"`

	// Region to use for IAM API calls.
	//
	// IAM is global so the calls are not regional but the Region will specify
	// which region the API calls are sent to.
	Region *string

	// The name of the role to associate the policy with.
	RoleName string `input:"role_name"`

	// No outputs
}

// Type returns the type name for an AWS IAM role policy resource.
func (IAMRolePolicy) Type() string { return "aws_iam_role_policy" }

// Create attaches an inline role policy to and IAM role.
func (p *IAMRolePolicy) Create(ctx context.Context, r *resource.CreateRequest) error {
	svc, err := iamService(r.Auth, p.Region)
	if err != nil {
		return errors.Wrap(err, "get client")
	}

	req := svc.PutRolePolicyRequest(&iam.PutRolePolicyInput{
		PolicyDocument: aws.String(p.PolicyDocument),
		PolicyName:     aws.String(p.PolicyName),
		RoleName:       aws.String(p.RoleName),
	})
	if _, err := req.Send(ctx); err != nil {
		return errors.Wrap(err, "send request")
	}

	// No outputs in response

	return nil
}

// Delete removes an inline role policy from an IAM role.
func (p *IAMRolePolicy) Delete(ctx context.Context, r *resource.DeleteRequest) error {
	svc, err := iamService(r.Auth, p.Region)
	if err != nil {
		return errors.Wrap(err, "get client")
	}

	req := svc.DeleteRolePolicyRequest(&iam.DeleteRolePolicyInput{
		PolicyName: aws.String(p.PolicyName),
		RoleName:   aws.String(p.RoleName),
	})
	if _, err := req.Send(ctx); err != nil {
		return errors.Wrap(err, "send request")
	}

	return nil
}

// Update removes the old role policy and attaches a new one.
func (p *IAMRolePolicy) Update(ctx context.Context, r *resource.UpdateRequest) error {
	// Does not support update, must do delete/create

	// Delete previous
	prev := r.Previous.(*IAMRolePolicy)
	if err := prev.Delete(ctx, r.DeleteRequest()); err != nil {
		return errors.Wrap(err, "update-delete")
	}

	// Create next
	if err := p.Create(ctx, r.CreateRequest()); err != nil {
		return errors.Wrap(err, "update-create")
	}

	return nil
}
