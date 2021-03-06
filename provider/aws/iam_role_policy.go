package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/cenkalti/backoff"
	"github.com/func/func/resource"
)

// IAMRolePolicy is an inline role policy, attached to a role.
type IAMRolePolicy struct {
	// Inputs

	// The policy document.
	PolicyDocument string `func:"input"`

	// The name of the policy document.
	PolicyName string `func:"input"`

	// Region to use for IAM API calls.
	//
	// IAM is global so the calls are not regional but the Region will specify
	// which region the API calls are sent to.
	Region *string `func:"input"`

	// The name of the role to associate the policy with.
	RoleName string `func:"input"`

	// No outputs

	iamService
}

// Create attaches an inline role policy to and IAM role.
func (p *IAMRolePolicy) Create(ctx context.Context, r *resource.CreateRequest) error {
	svc, err := p.service(r.Auth, p.Region)
	if err != nil {
		return err
	}

	input := &iam.PutRolePolicyInput{
		PolicyDocument: aws.String(p.PolicyDocument),
		PolicyName:     aws.String(p.PolicyName),
		RoleName:       aws.String(p.RoleName),
	}
	if err := input.Validate(); err != nil {
		return backoff.Permanent(err)
	}

	resp, err := svc.PutRolePolicyRequest(input).Send(ctx)
	if err != nil {
		return handlePutError(err)
	}

	// No outputs in response
	_ = resp

	return nil
}

// Delete removes an inline role policy from an IAM role.
func (p *IAMRolePolicy) Delete(ctx context.Context, r *resource.DeleteRequest) error {
	svc, err := p.service(r.Auth, p.Region)
	if err != nil {
		return err
	}

	input := &iam.DeleteRolePolicyInput{
		PolicyName: aws.String(p.PolicyName),
		RoleName:   aws.String(p.RoleName),
	}
	if err := input.Validate(); err != nil {
		return backoff.Permanent(err)
	}

	_, err = svc.DeleteRolePolicyRequest(input).Send(ctx)
	return handleDelError(err)
}

// Update removes the old role policy and attaches a new one.
func (p *IAMRolePolicy) Update(ctx context.Context, r *resource.UpdateRequest) error {
	// Does not support update, must do delete/create

	// Delete previous
	prev := r.Previous.(*IAMRolePolicy)
	if err := prev.Delete(ctx, r.DeleteRequest()); err != nil {
		return err
	}

	// Create next
	if err := p.Create(ctx, r.CreateRequest()); err != nil {
		return err
	}

	return nil
}
