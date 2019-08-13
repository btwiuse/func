package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/cenkalti/backoff"
	"github.com/func/func/resource"
)

// IAMRolePolicyAttachment attaches a role policy to a role.
//
// The same policy can be attached to many roles.
type IAMRolePolicyAttachment struct {
	// Inputs

	// The Amazon Resource Name (ARN) of the IAM policy you want to attach.
	//
	// For more information about ARNs, see Amazon Resource Names (ARNs) and AWS
	// Service Namespaces (http://docs.aws.amazon.com/general/latest/gr/aws-arns-and-namespaces.html)
	// in the AWS General Reference.
	PolicyARN string `func:"input"`

	// Region to use for IAM API calls.
	//
	// IAM is global so the calls are not regional but the Region will specify
	// which region the API calls are sent to.
	Region *string `func:"input"`

	// The name (friendly name, not ARN) of the role to attach the policy to.
	//
	// This parameter allows (through its regex pattern (http://wikipedia.org/wiki/regex))
	// a string of characters consisting of upper and lowercase alphanumeric characters
	// with no spaces. You can also include any of the following characters: _+=,.@-
	RoleName string `func:"input"`

	// No outputs

	iamService
}

// Create attaches a policy to a role.
func (p *IAMRolePolicyAttachment) Create(ctx context.Context, r *resource.CreateRequest) error {
	svc, err := p.service(r.Auth, p.Region)
	if err != nil {
		return err
	}

	input := &iam.AttachRolePolicyInput{
		PolicyArn: aws.String(p.PolicyARN),
		RoleName:  aws.String(p.RoleName),
	}
	if err := input.Validate(); err != nil {
		return backoff.Permanent(err)
	}

	resp, err := svc.AttachRolePolicyRequest(input).Send(ctx)
	if err != nil {
		return handlePutError(err)
	}

	// No outputs in response
	_ = resp

	return nil
}

// Delete removes a policy attachment.
func (p *IAMRolePolicyAttachment) Delete(ctx context.Context, r *resource.DeleteRequest) error {
	svc, err := p.service(r.Auth, p.Region)
	if err != nil {
		return err
	}

	input := &iam.DetachRolePolicyInput{
		PolicyArn: aws.String(p.PolicyARN),
		RoleName:  aws.String(p.RoleName),
	}
	if err := input.Validate(); err != nil {
		return backoff.Permanent(err)
	}

	_, err = svc.DetachRolePolicyRequest(input).Send(ctx)
	return handleDelError(err)
}

// Update removes the previous attachment and creates a new one.
func (p *IAMRolePolicyAttachment) Update(ctx context.Context, r *resource.UpdateRequest) error {
	// Does not support update, must do delete/create

	// Delete previous
	prev := r.Previous.(*IAMRolePolicyAttachment)
	if err := prev.Delete(ctx, r.DeleteRequest()); err != nil {
		return err
	}

	// Create next
	if err := p.Create(ctx, r.CreateRequest()); err != nil {
		return err
	}

	return nil
}
