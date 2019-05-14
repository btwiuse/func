package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/func/func/resource"
	"github.com/pkg/errors"
)

// LambdaInvokePermission sets permissions on a Lambda function.
type LambdaInvokePermission struct {
	// Inputs

	// The AWS Lambda action you want to allow in this statement. Each Lambda action
	// is a string starting with lambda: followed by the API name . For example,
	// lambda:CreateFunction. You can use wildcard (lambda:*) to grant permission
	// for all AWS Lambda actions.
	Action *string `func:"input,required"`

	// A unique token that must be supplied by the principal invoking the function.
	// This is currently only used for Alexa Smart Home functions.
	EventSourceToken *string `func:"input"`

	// The name of the Lambda function.
	//
	// Name formats
	//
	//  - Function name - `MyFunction`.
	//  - Function ARN - `arn:aws:lambda:us-west-2:123456789012:function:MyFunction`.
	//  - Partial ARN - `123456789012:function:MyFunction`.
	//
	// The length constraint applies only to the full ARN. If you specify only
	// the function name, it is limited to 64 characters in length.
	FunctionName *string `func:"input,required"`

	// The principal who is getting this permission. The principal can be an
	// AWS service (e.g. `s3.amazonaws.com` or `sns.amazonaws.com`) for service
	// triggers, or an account ID for cross-account access. If you specify a
	// service as a principal, use the SourceArn parameter to limit who can
	// invoke the function through that service.
	Principal *string `func:"input,required"`

	// Region the Lambda function has been deployed to.
	Region string `func:"input,required"`

	// Specify a version or alias to add permissions to a published version of the
	// function.
	Qualifier *string `func:"input"`

	// An optional value you can use to ensure you are updating the latest update
	// of the function version or alias. If the RevisionID you pass doesn't match
	// the latest RevisionID of the function or alias, it will fail with an error
	// message.
	RevisionID *string `func:"input"`

	// This parameter is used for S3 and SES. The AWS account ID (without a hyphen)
	// of the source owner. For example, if the SourceArn identifies a bucket, then
	// this is the bucket owner's account ID. You can use this additional condition
	// to ensure the bucket you specify is owned by a specific account (it is possible
	// the bucket owner deleted the bucket and some other AWS account created the
	// bucket). You can also use this condition to specify all sources (that is,
	// you don't specify the SourceArn) owned by a specific account.
	SourceAccount *string `func:"input"`

	// The Amazon Resource Name of the invoker.
	//
	// If you add a permission to a service principal without providing the source
	// ARN, any AWS account that creates a mapping to your function ARN can invoke
	// your Lambda function.
	SourceARN *string `func:"input" validate:"arn"`

	// A unique statement identifier.
	StatementID *string `func:"input,required"`

	// Outputs

	// The permission statement you specified in the request. The response returns
	// the same as a string using a backslash ("\") as an escape character in the
	// JSON.
	Statement *string `func:"output"`

	lambdaService
}

// Create creates an AWS lambda function.
func (p *LambdaInvokePermission) Create(ctx context.Context, r *resource.CreateRequest) error {
	svc, err := p.service(r.Auth, p.Region)
	if err != nil {
		return errors.Wrap(err, "get client")
	}

	input := &lambda.AddPermissionInput{
		Action:           p.Action,
		EventSourceToken: p.EventSourceToken,
		FunctionName:     p.FunctionName,
		Principal:        p.Principal,
		Qualifier:        p.Qualifier,
		RevisionId:       p.RevisionID,
		SourceAccount:    p.SourceAccount,
		SourceArn:        p.SourceARN,
		StatementId:      p.StatementID,
	}

	req := svc.AddPermissionRequest(input)
	resp, err := req.Send(ctx)
	if err != nil {
		return err
	}

	p.Statement = resp.Statement

	return nil
}

// Delete deletes the lambda function.
func (p *LambdaInvokePermission) Delete(ctx context.Context, r *resource.DeleteRequest) error {
	svc, err := p.service(r.Auth, p.Region)
	if err != nil {
		return errors.Wrap(err, "get client")
	}

	req := svc.RemovePermissionRequest(&lambda.RemovePermissionInput{
		FunctionName: p.FunctionName,
		Qualifier:    p.Qualifier,
		RevisionId:   p.RevisionID,
		StatementId:  p.StatementID,
	})
	_, err = req.Send(ctx)
	return err
}

// Update updates the lambda function.
func (p *LambdaInvokePermission) Update(ctx context.Context, r *resource.UpdateRequest) error {
	// Permission cannot be updated, delete and create nwe

	prev := r.Previous.(*LambdaInvokePermission)

	if err := prev.Delete(ctx, r.DeleteRequest()); err != nil {
		return errors.Wrap(err, "update-delete")
	}
	if err := p.Create(ctx, r.CreateRequest()); err != nil {
		return errors.Wrap(err, "update-create")
	}

	return nil
}
