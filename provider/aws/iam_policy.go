package aws

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/cenkalti/backoff"
	"github.com/func/func/resource"
	"github.com/pkg/errors"
)

// IAMPolicy describes a policy.
//
// The policy can be attached to a role using `aws_iam_role_policy_attachment`.
type IAMPolicy struct {
	// Inputs

	// A friendly description of the policy.
	//
	// Typically used to store information about the permissions defined in the
	// policy. For example, `Grants access to production DynamoDB tables.`
	//
	// The policy description is immutable. After a value is assigned, it cannot
	// be changed.
	Description *string `func:"input"`

	// The path for the policy.
	//
	// For more information about paths, see
	// [IAM Identifiers](http://docs.aws.amazon.com/IAM/latest/UserGuide/Using_Identifiers.html)
	// in the IAM User Guide.
	//
	// If the path is not set, it defaults to a slash (`/`).
	Path *string `func:"input"`

	// The JSON policy document that you want to use as the content for the new
	// policy.
	PolicyDocument string `func:"input"`

	// The friendly name of the policy.
	PolicyName string `func:"input"`

	// Region to use for IAM API calls.
	//
	// IAM is global so the calls are not regional but the Region will specify
	// which region the API calls are sent to.
	Region *string `func:"input"`

	// Outputs

	// The Amazon Resource Name (ARN). ARNs are unique identifiers for AWS resources.
	//
	// For more information about ARNs, go to Amazon Resource Names (ARNs) and AWS
	// Service Namespaces (http://docs.aws.amazon.com/general/latest/gr/aws-arns-and-namespaces.html)
	// in the AWS General Reference.
	ARN *string `func:"output"`

	// The number of entities (users, groups, and roles) that the policy is attached
	// to.
	AttachmentCount *int64 `func:"output"`

	// RFC3339 formatted date and time when the policy was created.
	CreateDate string `func:"output"`

	// The identifier for the version of the policy that is set as the default version.
	DefaultVersionID *string `func:"output"`

	// Specifies whether the policy can be attached to an IAM user, group, or role.
	IsAttachable *bool `func:"output"`

	// The number of entities (users and roles) for which the policy is used to
	// set the permissions boundary.
	//
	// For more information about permissions boundaries, see Permissions Boundaries
	// for IAM Identities
	// (http://docs.aws.amazon.com/IAM/latest/UserGuide/access_policies_boundaries.html)
	// in the IAM User Guide.
	PermissionsBoundaryUsageCount *int64 `func:"output"`

	// The stable and unique string identifying the policy.
	//
	// For more information about IDs, see IAM Identifiers
	// (http://docs.aws.amazon.com/IAM/latest/UserGuide/Using_Identifiers.html)
	// in the Using IAM guide.
	PolicyID *string `func:"output"`

	// The date and time when the policy was last updated.
	//
	// When a policy has only one version, this field contains the date and time
	// when the policy was created. When a policy has more than one version, this
	// field contains the date and time when the most recent policy version was
	// created.
	UpdateDate string `func:"output"`

	iamService
}

// Create creates a new IAM policy.
func (p *IAMPolicy) Create(ctx context.Context, r *resource.CreateRequest) error {
	svc, err := p.service(r.Auth, p.Region)
	if err != nil {
		return err
	}

	input := &iam.CreatePolicyInput{
		Description:    p.Description,
		Path:           p.Path,
		PolicyDocument: aws.String(p.PolicyDocument),
		PolicyName:     aws.String(p.PolicyName),
	}
	if err := input.Validate(); err != nil {
		return backoff.Permanent(err)
	}

	resp, err := svc.CreatePolicyRequest(input).Send(ctx)
	if err != nil {
		return handlePutError(err)
	}

	p.ARN = resp.Policy.Arn
	p.AttachmentCount = resp.Policy.AttachmentCount
	p.CreateDate = resp.Policy.CreateDate.Format(time.RFC3339)
	p.DefaultVersionID = resp.Policy.DefaultVersionId
	p.IsAttachable = resp.Policy.IsAttachable
	p.PermissionsBoundaryUsageCount = resp.Policy.PermissionsBoundaryUsageCount
	p.PolicyID = resp.Policy.PolicyId
	p.UpdateDate = resp.Policy.UpdateDate.Format(time.RFC3339)

	return nil
}

// Delete deletes the IAM policy.
func (p *IAMPolicy) Delete(ctx context.Context, r *resource.DeleteRequest) error {
	svc, err := p.service(r.Auth, p.Region)
	if err != nil {
		return err
	}

	input := &iam.DeletePolicyInput{
		PolicyArn: p.ARN,
	}
	if err := input.Validate(); err != nil {
		return backoff.Permanent(err)
	}

	_, err = svc.DeletePolicyRequest(input).Send(ctx)
	return handleDelError(err)
}

// Update returns an error. A policy cannot be updated.
func (p *IAMPolicy) Update(ctx context.Context, r *resource.UpdateRequest) error {
	// NOTE: We could delete and create the policy. However this may cause
	// problems in case the policy is attached somewhere. For now, return an
	// error instead.
	return backoff.Permanent(errors.New("policy cannot be updated"))
}
