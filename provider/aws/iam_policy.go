//nolint: lll
//go:generate go run ../../tools/structdoc/main.go --file $GOFILE --struct IAMPolicy --template ../../tools/structdoc/template.txt --data type=aws_iam_policy --output ../../docs/resources/aws/iam_policy.md

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
	Description *string `input:"description"`

	// The path for the policy.
	//
	// For more information about paths, see
	// [IAM Identifiers](http://docs.aws.amazon.com/IAM/latest/UserGuide/Using_Identifiers.html)
	// in the IAM User Guide.
	//
	// If the path is not set, it defaults to a slash (`/`).
	Path *string `input:"path"`

	// The JSON policy document that you want to use as the content for the new
	// policy.
	PolicyDocument string `input:"policy_document"`

	// The friendly name of the policy.
	PolicyName string `input:"policy_name"`

	// Region to use for IAM API calls.
	//
	// IAM is global so the calls are not regional but the Region will specify
	// which region the API calls are sent to.
	Region *string

	// Outputs

	// The Amazon Resource Name (ARN). ARNs are unique identifiers for AWS resources.
	//
	// For more information about ARNs, go to Amazon Resource Names (ARNs) and AWS
	// Service Namespaces (http://docs.aws.amazon.com/general/latest/gr/aws-arns-and-namespaces.html)
	// in the AWS General Reference.
	ARN string `output:"arn"`

	// The number of entities (users, groups, and roles) that the policy is attached
	// to.
	AttachmentCount int64 `output:"attachment_count"`

	// The date and time when the policy was created.
	CreateDate time.Time `output:"create_date"`

	// The identifier for the version of the policy that is set as the default version.
	DefaultVersionID string `output:"default_version_id"`

	// Specifies whether the policy can be attached to an IAM user, group, or role.
	IsAttachable bool `output:"is_attachable"`

	// The number of entities (users and roles) for which the policy is used to
	// set the permissions boundary.
	//
	// For more information about permissions boundaries, see Permissions Boundaries
	// for IAM Identities
	// (http://docs.aws.amazon.com/IAM/latest/UserGuide/access_policies_boundaries.html)
	// in the IAM User Guide.
	PermissionsBoundaryUsageCount int64 `output:"permissions_boundary_usage_count"`

	// The stable and unique string identifying the policy.
	//
	// For more information about IDs, see IAM Identifiers
	// (http://docs.aws.amazon.com/IAM/latest/UserGuide/Using_Identifiers.html)
	// in the Using IAM guide.
	PolicyID string `output:"policy_id"`

	// The date and time when the policy was last updated.
	//
	// When a policy has only one version, this field contains the date and time
	// when the policy was created. When a policy has more than one version, this
	// field contains the date and time when the most recent policy version was
	// created.
	UpdateDate time.Time `output:"update_date"`

	iamService
}

// Type returns the type name for an AWS IAM policy resource.
func (p *IAMPolicy) Type() string { return "aws_iam_policy" }

// Create creates a new IAM policy.
func (p *IAMPolicy) Create(ctx context.Context, r *resource.CreateRequest) error {
	svc, err := p.service(r.Auth, p.Region)
	if err != nil {
		return errors.Wrap(err, "get client")
	}

	req := svc.CreatePolicyRequest(&iam.CreatePolicyInput{
		Description:    p.Description,
		Path:           p.Path,
		PolicyDocument: aws.String(p.PolicyDocument),
		PolicyName:     aws.String(p.PolicyName),
	})
	resp, err := req.Send(ctx)
	if err != nil {
		return errors.Wrap(err, "send request")
	}

	p.ARN = *resp.Policy.Arn
	p.AttachmentCount = *resp.Policy.AttachmentCount
	p.CreateDate = *resp.Policy.CreateDate
	p.DefaultVersionID = *resp.Policy.DefaultVersionId
	p.IsAttachable = *resp.Policy.IsAttachable
	p.PermissionsBoundaryUsageCount = *resp.Policy.PermissionsBoundaryUsageCount
	p.PolicyID = *resp.Policy.PolicyId
	p.UpdateDate = *resp.Policy.UpdateDate

	return nil
}

// Delete deletes the IAM policy.
func (p *IAMPolicy) Delete(ctx context.Context, r *resource.DeleteRequest) error {
	svc, err := p.service(r.Auth, p.Region)
	if err != nil {
		return errors.Wrap(err, "get client")
	}

	req := svc.DeletePolicyRequest(&iam.DeletePolicyInput{
		PolicyArn: aws.String(p.ARN),
	})
	if _, err := req.Send(ctx); err != nil {
		return errors.Wrap(err, "send request")
	}

	return nil
}

// Update returns an error. A policy cannot be updated.
func (p *IAMPolicy) Update(ctx context.Context, r *resource.UpdateRequest) error {
	// NOTE: We could delete and create the policy. However this may cause
	// problems in case the policy is attached somewhere. For now, return an
	// error instead.
	return backoff.Permanent(errors.New("policy cannot be updated"))
}
