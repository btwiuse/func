//nolint: lll
//go:generate go run ../../tools/structdoc/main.go --file $GOFILE --struct IAMRolePolicyAttachment --template ../../tools/structdoc/template.txt --data type=aws_iam_role_policy_attachment --output ../../docs/resources/aws/iam_role_policy_attachment.md

package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/iamiface"
	"github.com/func/func/provider/aws/internal/config"
	"github.com/func/func/resource"
	"github.com/pkg/errors"
)

// IAMRolePolicyAttachment attaches a role policy to a role.
//
// The same policy can be attached to many roles.
type IAMRolePolicyAttachment struct {
	// The Amazon Resource Name (ARN) of the IAM policy you want to attach.
	//
	// For more information about ARNs, see Amazon Resource Names (ARNs) and AWS
	// Service Namespaces (http://docs.aws.amazon.com/general/latest/gr/aws-arns-and-namespaces.html)
	// in the AWS General Reference.
	PolicyARN string `input:"policy_arn"`

	// The name (friendly name, not ARN) of the role to attach the policy to.
	//
	// This parameter allows (through its regex pattern (http://wikipedia.org/wiki/regex))
	// a string of characters consisting of upper and lowercase alphanumeric characters
	// with no spaces. You can also include any of the following characters: _+=,.@-
	RoleName string `input:"role_name"`

	// No outputs

	svc iamiface.IAMAPI
}

// Type returns the type name for an AWS IAM policy attachment.
func (p *IAMRolePolicyAttachment) Type() string { return "aws_iam_role_policy_attachment" }

// Create attaches a policy to a role.
func (p *IAMRolePolicyAttachment) Create(ctx context.Context, r *resource.CreateRequest) error {
	svc, err := p.service(r.Auth)
	if err != nil {
		return errors.Wrap(err, "get client")
	}

	req := svc.AttachRolePolicyRequest(&iam.AttachRolePolicyInput{
		PolicyArn: aws.String(p.PolicyARN),
		RoleName:  aws.String(p.RoleName),
	})
	req.SetContext(ctx)
	if _, err := req.Send(); err != nil {
		return errors.Wrap(err, "send request")
	}

	// No outputs in response

	return nil
}

// Delete removes a policy attachment.
func (p *IAMRolePolicyAttachment) Delete(ctx context.Context, r *resource.DeleteRequest) error {
	svc, err := p.service(r.Auth)
	if err != nil {
		return errors.Wrap(err, "get client")
	}

	req := svc.DetachRolePolicyRequest(&iam.DetachRolePolicyInput{
		PolicyArn: aws.String(p.PolicyARN),
		RoleName:  aws.String(p.RoleName),
	})
	req.SetContext(ctx)
	if _, err := req.Send(); err != nil {
		return errors.Wrap(err, "send request")
	}

	return nil
}

// Update removes the previous attachment and creates a new one.
func (p *IAMRolePolicyAttachment) Update(ctx context.Context, r *resource.UpdateRequest) error {
	// Does not support update, must do delete/create

	// Delete previous
	prev := r.Previous.(*IAMRolePolicyAttachment)
	if err := prev.Delete(ctx, nil); err != nil {
		return errors.Wrap(err, "update-delete")
	}

	// Create next
	if err := p.Create(ctx, nil); err != nil {
		return errors.Wrap(err, "update-create")
	}

	return nil
}

func (p *IAMRolePolicyAttachment) service(auth resource.AuthProvider) (iamiface.IAMAPI, error) {
	if p.svc == nil {
		cfg, err := config.DefaultRegion(auth)
		if err != nil {
			return nil, errors.Wrap(err, "get aws config")
		}
		p.svc = iam.New(cfg)
	}
	return p.svc, nil
}
