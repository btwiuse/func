//nolint: lll
//go:generate go run ../../tools/structdoc/main.go --file $GOFILE --struct STSCallerIdentity --template ../../tools/structdoc/template.txt --data type=aws_sts_caller_identity --output ../../docs/resources/aws/sts_caller_identity.md

package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/func/func/resource"
	"github.com/pkg/errors"
)

// STSCallerIdentity returns info for the current user.
type STSCallerIdentity struct {
	// Outputs

	// The AWS account ID number of the account that owns or contains the calling
	// entity.
	Account string `output:"account"`

	// The AWS ARN associated with the calling entity.
	ARN string `output:"arn"`

	// Region to use for STS API calls.
	//
	// STS is global so the calls are not regional but the Region will specify
	// which region the API calls are sent to.
	Region *string

	// The unique identifier of the calling entity. The exact value depends on
	// the type of entity making the call. The values returned are those listed
	// in the aws:userid column in the
	// [Principal table](http://docs.aws.amazon.com/IAM/latest/UserGuide/reference_policies_variables.html#principaltable)
	// found on the Policy Variables reference page in the IAM User Guide.
	UserID string `output:"user_id"`
}

// Type returns the type name for an AWS IAM policy resource.
func (p *STSCallerIdentity) Type() string { return "aws_sts_caller_identity" }

// Create reads the current caller identity
func (p *STSCallerIdentity) Create(ctx context.Context, r *resource.CreateRequest) error {
	svc, err := stsService(r.Auth, p.Region)
	if err != nil {
		return errors.Wrap(err, "get client")
	}

	req := svc.GetCallerIdentityRequest(&sts.GetCallerIdentityInput{})
	req.SetContext(ctx)
	resp, err := req.Send()
	if err != nil {
		return err
	}

	p.Account = *resp.Account
	p.ARN = *resp.Arn
	p.UserID = *resp.UserId

	return nil
}

// Delete deletes the IAM policy.
func (p *STSCallerIdentity) Delete(ctx context.Context, r *resource.DeleteRequest) error {
	return nil
}

// Update returns an error. A policy cannot be updated.
func (p *STSCallerIdentity) Update(ctx context.Context, r *resource.UpdateRequest) error {
	return nil
}
